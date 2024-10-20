package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/ambi/grpcapps/proto/echo"
)

var logger *slog.Logger
var grpcClient pb.EchoServiceClient

type EchoResponse struct {
	Message string `json:"message"`
}

func clientInterceptor(logger *slog.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var traceID string
		if md, ok := metadata.FromOutgoingContext(ctx); ok {
			if values := md.Get("trace-id"); len(values) > 0 {
				traceID = values[0]
			}
		}
		if traceID == "" {
			traceID = uuid.New().String()
			ctx = metadata.AppendToOutgoingContext(ctx, "trace-id", traceID)
		}

		// リクエストログ
		logger.Info("gRPC client request",
			"trace_id", traceID,
			"method", method,
			"request", req,
		)

		// gRPC呼び出し
		err := invoker(ctx, method, req, reply, cc, opts...)

		// レスポンスログ
		logger.Info("gRPC client response",
			"trace_id", traceID,
			"method", method,
			"response", reply,
			"error", err,
		)

		return err
	}
}

func echo(resp http.ResponseWriter, req *http.Request) {
	traceID := uuid.New().String()
	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("trace-id", traceID))

	logger.Info("echo start", "trace-id", traceID)
	defer logger.Info("echo end", "trace-id", traceID)

	msg := req.URL.Query().Get("message")

	result, err := grpcClient.Echo(ctx, &pb.EchoRequest{Message: msg})
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.Header().Set("Content-Type", "application/json; charset=utf-8")

	b, err := json.Marshal(&EchoResponse{Message: result.Message})
	if err != nil {
		log.Println(err)

		resp.WriteHeader(http.StatusInternalServerError)
		s := `{"status": "System Error"}`
		if _, err := resp.Write([]byte(s)); err != nil {
			log.Println(err)
		}
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(b); err != nil {
		log.Println(err)
	}
}

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(clientInterceptor(logger)))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	grpcClient = pb.NewEchoServiceClient(conn)

	server := http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	http.Handle("/echo", http.HandlerFunc(echo))

	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to listen and serve: %v", err)
	}
}

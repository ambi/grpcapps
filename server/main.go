package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pb "github.com/ambi/grpcapps/proto/echo"
)

var logger *slog.Logger

func serverInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// メタデータからトレースIDを取得
		var traceID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("trace-id"); len(values) > 0 {
				traceID = values[0]
			}
		}

		// リクエストログ
		logger.Info("gRPC server request",
			"trace_id", traceID,
			"method", info.FullMethod,
			"request", req,
		)

		// 実際のハンドラーの呼び出し
		resp, err := handler(ctx, req)

		// レスポンスログ
		logger.Info("gRPC server response",
			"trace_id", traceID,
			"method", info.FullMethod,
			"response", resp,
			"error", err,
		)

		return resp, err
	}
}

type server struct {
	pb.UnimplementedEchoServiceServer
}

func (s *server) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	logger.Info("echo start")
	defer logger.Info("echo end")

	return &pb.EchoResponse{
		Message: req.Message,
	}, nil
}

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(serverInterceptor(logger)))
	pb.RegisterEchoServiceServer(s, &server{})

	log.Println("Starting gRPC server on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

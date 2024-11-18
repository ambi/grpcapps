```shell
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/echo/echo.proto

grpc_tools_ruby_protoc -I proto --ruby_out=ruby/pb --grpc_out=ruby/pb proto/echo/echo.proto
```

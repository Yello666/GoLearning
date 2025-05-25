package main

import (
	"context"
	"fmt"
	pb "github.com/Yello666/GoLearning/goPrctice/gRPC/server/RPC"
	"google.golang.org/grpc"
)

func main() {
	conn, _ := grpc.Dial("localhost:8080", grpc.WithInsecure())
	client := pb.NewMyServiceClient(conn)                                            // 使用 _grpc.pb.go 生成的客户端
	res, _ := client.SayHello(context.Background(), &pb.HelloRequest{Name: "Hello"}) // 使用 service.pb.go 生成的 Request
	fmt.Println(res.RspMessage)                                                      // 使用 service.pb.go 生成的方法
}

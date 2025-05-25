package main

import (
	"context"
	"fmt"
	pb "github.com/Yello666/GoLearning/goPrctice/gRPC/server/RPC"
	"google.golang.org/grpc"
	"net"
)

type server struct {
	pb.UnimplementedMyServiceServer
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{RspMessage: "hello" + req.Name}, nil
}
func main() {
	listen, _ := net.Listen("tcp", ":8080")
	grpcServer := grpc.NewServer()
	pb.RegisterMyServiceServer(grpcServer, &server{})
	err := grpcServer.Serve(listen)
	if err != nil {
		fmt.Println("error in sevice running:%v", err)
		return
	}
}

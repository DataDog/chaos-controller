// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	pb "github.com/DataDog/chaos-controller/dogfood/chaosdogfood"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var serverAddr string

func init() {
	var serverPort int

	var serverIP string

	flag.IntVar(&serverPort, "server_port", 50000, "Port where gRPC server is running")
	flag.StringVar(&serverIP, "server_ip", "", "IP address where gRPC server is hosted")
	flag.Parse()

	serverAddr = fmt.Sprintf("%s:%d", serverIP, serverPort)
}

type chaosDogfoodServer struct {
	pb.UnimplementedChaosDogfoodServer
}

func (s *chaosDogfoodServer) Order(ctx context.Context, req *pb.FoodRequest) (*pb.FoodReply, error) {
	fmt.Printf("| %v food ordered\n", req.Animal)
	return &pb.FoodReply{Message: "Mock Reply", ConfirmationId: 1}, nil
}

func (s *chaosDogfoodServer) GetCatalog(ctx context.Context, req *emptypb.Empty) (*pb.CatalogReply, error) {
	fmt.Println("x\n| catalog delivered")
	return &pb.CatalogReply{Items: []*pb.CatalogItem{}}, nil
}

func main() {
	fmt.Printf("listening on %v...\n", serverAddr)

	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	pb.RegisterChaosDogfoodServer(s, &chaosDogfoodServer{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

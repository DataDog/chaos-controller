// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"context"
	"errors"
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

var catalog = map[string]string{
	"cat": "Meowmix",
	"dog": "Chewey",
	"cow": "Grassfed",
}

type chaosDogfoodServer struct {
	pb.UnimplementedChaosDogfoodServer
	replyCounter int32
}

func (s *chaosDogfoodServer) Order(ctx context.Context, req *pb.FoodRequest) (*pb.FoodReply, error) {
	s.replyCounter++

	if food, ok := catalog[req.Animal]; ok {
		fmt.Printf("| proccessed order - %s\n", req.String())

		return &pb.FoodReply{
			Message:        fmt.Sprintf("%s is on its way!", food),
			ConfirmationId: s.replyCounter,
		}, nil
	}

	fmt.Printf("| * DECLINED ORDER - %s\n", req.String())

	return nil, errors.New("Sorry, we don't deliver food for your " + req.Animal + " =(")
}

func (s *chaosDogfoodServer) GetCatalog(ctx context.Context, req *emptypb.Empty) (*pb.CatalogReply, error) {
	fmt.Println("x\n| returned catalog")

	items := make([]*pb.CatalogItem, 0, len(catalog))

	for animal, food := range catalog {
		items = append(items, &pb.CatalogItem{
			Animal: animal,
			Food:   food,
		})
	}

	return &pb.CatalogReply{Items: items}, nil
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

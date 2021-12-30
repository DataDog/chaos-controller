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

	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"

	df_pb "github.com/DataDog/chaos-controller/dogfood/chaosdogfood"
	disruption_service "github.com/DataDog/chaos-controller/grpc"
	dl_pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	zaplog "github.com/DataDog/chaos-controller/log"
)

const chaosEnabled = true // In your application, make this a feature flag

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

type chaosDogfoodService struct {
	df_pb.UnimplementedChaosDogfoodServer
	replyCounter int32
}

func (s *chaosDogfoodService) Order(ctx context.Context, req *df_pb.FoodRequest) (*df_pb.FoodReply, error) {
	s.replyCounter++

	if food, ok := catalog[req.Animal]; ok {
		fmt.Printf("| proccessed order - %s\n", req.String())

		return &df_pb.FoodReply{
			Message:        fmt.Sprintf("%s is on its way!", food),
			ConfirmationId: s.replyCounter,
		}, nil
	}

	fmt.Printf("| * DECLINED ORDER - %s\n", req.String())

	return nil, errors.New("Sorry, we don't deliver food for your " + req.Animal + " =(")
}

func (s *chaosDogfoodService) GetCatalog(ctx context.Context, req *emptypb.Empty) (*df_pb.CatalogReply, error) {
	fmt.Println("x\n| returned catalog")

	items := make([]*df_pb.CatalogItem, 0, len(catalog))

	for animal, food := range catalog {
		items = append(items, &df_pb.CatalogItem{
			Animal: animal,
			Food:   food,
		})
	}

	return &df_pb.CatalogReply{Items: items}, nil
}

func main() {
	fmt.Printf("listening on %v...\n", serverAddr)

	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		log.Fatalf("failed to listen: %s\n", err)
	}

	var dogfoodServer *grpc.Server

	// In your application, check the feature flag to decide if the interceptor should be used
	if chaosEnabled == true {
		fmt.Println("CHAOS ENABLED")

		disruptionLogger, error := zaplog.NewZapLogger()
		if error != nil {
			log.Fatal("error creating controller logger")
			return
		}

		disruptionListener := disruption_service.NewDisruptionListener(disruptionLogger)

		dogfoodServer = grpc.NewServer(
			grpc.UnaryInterceptor(disruptionListener.ChaosServerInterceptor),
		)

		df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})
		dl_pb.RegisterDisruptionListenerServer(dogfoodServer, disruptionListener)
	} else {
		dogfoodServer = grpc.NewServer()

		df_pb.RegisterChaosDogfoodServer(dogfoodServer, &chaosDogfoodService{})
	}

	if err := dogfoodServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

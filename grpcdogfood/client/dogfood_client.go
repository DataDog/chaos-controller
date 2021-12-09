// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	pb "github.com/DataDog/chaos-controller/grpcdogfood/chaosdogfood"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const serverAddr = "localhost:50051"

func orderWithTimeout(client pb.ChaosDogfoodClient, animal string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Order(ctx, &pb.FoodRequest{Animal: animal})
	if err != nil {
		return "", err
	}

	return res.Message, nil
}

func getCatalogWithTimeout(client pb.ChaosDogfoodClient) ([]*pb.CatalogItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.GetCatalog(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	return res.Items, nil
}

func main() {
	// create and eventually close connection
	fmt.Println("connecting to " + serverAddr + "...")

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Println("error closing gRPC connection to dogfood server", "error", err)
		}
	}()

	// generate and use client
	client := pb.NewChaosDogfoodClient(conn)

	for {
		items, err := getCatalogWithTimeout(client)
		if err != nil {
			fmt.Println("| ERROR getting catalog: " + err.Error())
		}

		fmt.Println("| got catalog: " + strconv.Itoa(len(items)) + " items returned")
		time.Sleep(time.Second)

		order, err := orderWithTimeout(client, "cat")
		if err != nil {
			fmt.Println("| ERROR ordering food: " + err.Error())
		}

		fmt.Println("| ordered: " + order)
		time.Sleep(time.Second)
	}
}

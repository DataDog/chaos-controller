// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

//go:build !cgo
// +build !cgo

package main

/*
#cgo LDFLAGS: -lelf -lz
#include <bpf/bpf.h>
*/
import "C"

import (
	goflag "flag"
	"fmt"
	"syscall"
	"unsafe"

	"github.com/DataDog/chaos-controller/log"
	"github.com/aquasecurity/libbpfgo"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
	err      error
	logger   *zap.SugaredLogger
	nMethods = flag.StringArray("method", []string{}, "Filter by http method: GET, DELETE, POST, PUT, HEAD, PATCH, CONNECT, OPTIONS or TRACE")
	nPaths   = flag.StringArray("path", []string{"/"}, "Filter by http path. Default: /")
)

const MaxPathLen = 90
const MaxMethodLen = 8
const PathsBPFMapName = "filter_paths"
const MethodsBPFMapName = "filter_methods"

func main() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()
	logger, err = log.NewZapLogger()
	if err != nil {
		logger.Fatalf("could not initialize the logger: %w", err, err)
	}

	if err := populateBPFMap(MethodsBPFMapName, *nMethods, MaxMethodLen); err != nil {
		logger.Fatalf("could not populate %s map: %w", MethodsBPFMapName, err)
	}

	logger.Info("the %s is successfully updated", MethodsBPFMapName)

	if err := populateBPFMap(PathsBPFMapName, *nPaths, MaxPathLen); err != nil {
		logger.Fatalf("could not populate %s map: %w", PathsBPFMapName, err)
	}

	logger.Info("the %s is successfully updated", PathsBPFMapName)
}

func populateBPFMap(mapName string, fields []string, fieldSize int) error {
	bpfMapIds, err := libbpfgo.GetMapsIDsByName(mapName)
	if err != nil {
		return fmt.Errorf("could not get maps ids of %s maps: %w", mapName, err)
	}

	bpfMap, err := libbpfgo.GetMapByID(bpfMapIds[0])
	if err != nil {
		return fmt.Errorf("could not get the %s map: %w", mapName, err)
	}

	for i, field := range fields {
		if err := updateMap(uint32(i), []byte(field), fieldSize, bpfMap); err != nil {
			closeMap(bpfMap)
			return fmt.Errorf("could not update the %d field with %s value %s map: %w", i, field, mapName, err)
		}
	}

	return closeMap(bpfMap)
}

func updateMap(key uint32, value []byte, valueSize int, bpfMap *libbpfgo.BPFMapLow) error {
	valueBytes := make([]byte, valueSize)
	copy(valueBytes[:len(value)], value)

	return bpfMap.Update(unsafe.Pointer(&key), unsafe.Pointer(&valueBytes[0]))
}

func closeMap(bpfMap *libbpfgo.BPFMapLow) error {
	if err := syscall.Close(bpfMap.FileDescriptor()); err != nil {
		return fmt.Errorf("failed to close the file descriptor of the %s map with %d id: %w", bpfMap.Name(), bpfMap.FileDescriptor(), err)
	}

	return nil
}

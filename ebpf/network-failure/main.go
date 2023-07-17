// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

//go:build !cgo
// +build !cgo

package main

import (
	"C"
	"flag"
	"github.com/DataDog/chaos-controller/log"
	"github.com/aquasecurity/libbpfgo"
	"go.uber.org/zap"
	"unsafe"
)

var (
	err     error
	logger  *zap.SugaredLogger
	nMethod = flag.String("m", "ALL", "Filter method")
	nPath   = flag.String("f", "/", "Filter path")
)

const ValueSize = 100
const MapName = "flags_map"

func main() {
	flag.Parse()
	path := []byte(*nPath)
	method := []byte(*nMethod)

	logger, err = log.NewZapLogger()
	if err != nil {
		logger.Fatalf("could not initialize the logger: %w", err, err)
	}

	bpfMap, err := libbpfgo.GetMapByName("flags_map")
	if err != nil {
		logger.Fatalf("could not get the flags_map: %w", err, err)
	}

	// Update the path
	if err = updateMap(uint32(0), path, ValueSize, bpfMap); err != nil {
		logger.Fatalf("could not update the path: %w", err)
	}

	// Update the method
	if err = updateMap(uint32(1), method, ValueSize, bpfMap); err != nil {
		logger.Fatalf("could not update the method: %w", err)
	}

	logger.Infof("the %s map is updated", MapName)
}

func updateMap(key uint32, value []byte, valueSize int, bpfMap *libbpfgo.BPFMap) error {
	valueBytes := make([]byte, valueSize)
	copy(valueBytes[:len(value)], value)

	logger.Debugf("UPDATE MAP %s key: %s, value: %s, value size: %d\n", bpfMap.GetName(), key, value, valueSize)

	return bpfMap.Update(unsafe.Pointer(&key), unsafe.Pointer(&valueBytes[0]))
}

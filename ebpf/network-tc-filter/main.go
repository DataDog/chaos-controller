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
	"flag"
	"fmt"
	"github.com/DataDog/chaos-controller/log"
	"go.uber.org/zap"
	"syscall"
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

type BPFMap struct {
	name string
	fd   C.int
}

func (b *BPFMap) Update(key, value unsafe.Pointer) error {
	errC := C.bpf_map_update_elem(b.fd, key, value, C.ulonglong(0))
	if errC != 0 {
		return fmt.Errorf("failed to update map %s: %w", b.name, syscall.Errno(-errC))
	}
	return nil
}

func main() {
	flag.Parse()
	path := []byte(*nPath)
	method := []byte(*nMethod)
	logger, err = log.NewZapLogger()
	if err != nil {
		logger.Fatalf("could not initialize the logger: %w", err, err)
	}

	bpfMap, err := GetMapByName("flags_map")
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

func updateMap(key uint32, value []byte, valueSize int, bpfMap *BPFMap) error {
	valueBytes := make([]byte, valueSize)
	copy(valueBytes[:len(value)], value)

	logger.Debugf("UPDATE MAP %s key: %s, value: %s, value size: %d\n", bpfMap.name, key, value, valueSize)

	return bpfMap.Update(unsafe.Pointer(&key), unsafe.Pointer(&valueBytes[0]))
}

func GetMapByName(name string) (*BPFMap, error) {
	startId := C.uint(0)
	nextId := C.uint(0)

	for {
		err := C.bpf_map_get_next_id(startId, &nextId)
		if err != 0 {
			return nil, fmt.Errorf("could not get the map: %w", syscall.Errno(-err))
		}

		startId = nextId + 1

		fd := C.bpf_map_get_fd_by_id(nextId)
		if fd < 0 {
			return nil, fmt.Errorf("could not get the file descriptor of %s", name)
		}

		info := C.struct_bpf_map_info{}
		infolen := C.uint(unsafe.Sizeof(info))
		err = C.bpf_obj_get_info_by_fd(fd, unsafe.Pointer(&info), &infolen)
		if err != 0 {
			return nil, fmt.Errorf("could not get the map info: %w", syscall.Errno(-err))
		}

		mapName := C.GoString((*C.char)(unsafe.Pointer(&info.name[0])))
		if mapName != name {
			continue
		}

		return &BPFMap{
			name: name,
			fd:   fd,
		}, nil
	}
	return nil, fmt.Errorf("the %s map does not exists", name)
}

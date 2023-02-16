// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

//go:build !cgo
// +build !cgo

package main

import (
	"C"
	"bytes"
	"encoding/binary"
	"flag"
	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/log"
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
	"go.uber.org/zap"
	"os"
	"os/signal"
)

var nFlag = flag.Uint64("p", 0, "Process to disrupt")
var nPath = flag.String("f", "/", "Filter path")

var logger *zap.SugaredLogger

func main() {
	// Defined a chanel to handle SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	var err error
	logger, err = log.NewZapLogger()
	must(err)

	// Create the bpf module
	bpfModule, err := bpf.NewModuleFromFile("/usr/local/bin/bpf-disk-failure.bpf.o")
	must(err)
	defer bpfModule.Close()

	initGlobalVariables(bpfModule)

	err = bpfModule.BPFLoadObject()
	must(err)

	// reads data from the trace pipe that bpf_trace_printk() writes to,
	// (/sys/kernel/debug/tracing/trace_pipe).
	go helpers.TracePipeListen()

	// Load the BPF program
	prog, err := bpfModule.GetProgram("injection_disk_failure")
	must(err)

	// Attach the kprope to catch sys openat syscall
	_, err = prog.AttachKprobe(ebpf.SysOpenat)
	must(err)

	// Create the ring buffer to store events
	e := make(chan []byte, 300)
	p, err := bpfModule.InitPerfBuf("events", e, nil, 1024)
	must(err)

	// Start the buffer
	p.Start()

	// Print events
	go func() {
		for data := range e {
			printEvent(data)
		}
	}()

	<-sig
	p.Stop()
}

func printEvent(data []byte) {
	ppid := int(binary.LittleEndian.Uint32(data[0:4]))
	pid := int(binary.LittleEndian.Uint32(data[4:8]))
	tid := int(binary.LittleEndian.Uint32(data[8:12]))
	gid := int(binary.LittleEndian.Uint32(data[12:16]))
	comm := string(bytes.TrimRight(data[16:], "\x00"))
	logger.Infof("Disrupt Ppid %d, Pid %d, Tid: %d, Gid: %d, Command: %s", ppid, pid, tid, gid, comm)
}

// The global variables are shared against the userspace application and the BPF application (loaded into the kernel).
// This global variables allow the user application to parametrise the BPF application.
func initGlobalVariables(bpfModule *bpf.Module) {
	flag.Parse()

	// Set the PID
	var pid uint32
	pid = uint32(*nFlag)
	if err := bpfModule.InitGlobalVariable("target_pid", pid); err != nil {
		must(err)
	}

	path := []byte(*nPath)
	if err := bpfModule.InitGlobalVariable("filter_path", path); err != nil {
		must(err)
	}

	currentPid := uint32(os.Getpid())
	if err := bpfModule.InitGlobalVariable("exclude_pid", currentPid); err != nil {
		must(err)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build !cgo
// +build !cgo

package main

import (
	"C"
	"bytes"
	"encoding/binary"
	"flag"
	"os"
	"os/signal"

	"github.com/DataDog/chaos-controller/log"
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
	"go.uber.org/zap"
)

var nPidNsInum = flag.Uint64("pid-ns-inum", 0, "PID namespace inode to disrupt (0 = all namespaces)")
var nPath = flag.String("path", "/", "Filter path")
var nProbability = flag.Uint64("probability", 100, "Probability to disrupt")
var nExitCode = flag.Uint64("exit-code", 1, "Exit code")

var logger *zap.SugaredLogger

func main() {
	// Defined a chanel to handle SIGINT
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	var err error
	logger, err = log.NewZapLogger()
	must(err)

	bpf.SetLoggerCbs(bpf.Callbacks{
		Log: func(level int, msg string) {
			switch level {
			case bpf.LibbpfDebugLevel:
				logger.Debug(msg)
			case bpf.LibbpfInfoLevel:
				logger.Info(msg)
			case bpf.LibbpfWarnLevel:
				logger.Warn(msg)
			default:
				logger.Error(msg)
			}
		},
	})

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

	// AttachGeneric attaches the fmod_ret program declared in the ELF section
	// (fmod_ret/__x64_sys_openat or fmod_ret/__arm64_sys_openat). fmod_ret
	// requires BPF trampoline support (Linux 5.7+). This intentionally
	// replaces the previous kprobe + bpf_override_return approach: fmod_ret
	// lets the program return a value directly without needing
	// bpf_override_return, which is unavailable on kernels built without
	// CONFIG_BPF_KPROBE_OVERRIDE. Nodes running kernels older than 5.7 are
	// not supported by this injector and will fail at BPFLoadObject() above.
	_, err = prog.AttachGeneric()
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
	pid := int(binary.LittleEndian.Uint32(data[0:4]))
	tid := int(binary.LittleEndian.Uint32(data[4:8]))
	gid := int(binary.LittleEndian.Uint32(data[8:12]))
	comm := string(bytes.TrimRight(data[12:], "\x00"))
	logger.Infof("Disrupt Pid %d, Tid: %d, Gid: %d, Command: %s", pid, tid, gid, comm)
}

// The global variables are shared against the userspace application and the BPF application (loaded into the kernel).
// This global variables allow the user application to parametrise the BPF application.
func initGlobalVariables(bpfModule *bpf.Module) {
	flag.Parse()

	pidNsInum := uint32(*nPidNsInum)
	if err := bpfModule.InitGlobalVariable("target_pid_ns_inum", pidNsInum); err != nil {
		must(err)
	}

	path := []byte(*nPath)
	if err := bpfModule.InitGlobalVariable("filter_path", path); err != nil {
		must(err)
	}

	exitCode := uint32(*nExitCode)
	if err := bpfModule.InitGlobalVariable("exit_code", exitCode); err != nil {
		must(err)
	}

	probability := uint32(*nProbability)
	if err := bpfModule.InitGlobalVariable("probability", probability); err != nil {
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

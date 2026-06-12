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
	"syscall"
	"unsafe"

	"github.com/DataDog/chaos-controller/ebpf"
	"github.com/DataDog/chaos-controller/log"
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
	"go.uber.org/zap"
)

var nPid = flag.Uint64("process", 0, "Process to disrupt")
var nCgroupPath = flag.String("cgroup-path", "", "Cgroupv2 directory path for ancestor-based filtering (covers kubectl exec sub-cgroups)")
var nFilterDirInode = flag.Uint64("filter-dir-inode", 0, "Inode of filter path parent directory; enables relative-path disruption (e.g. after cd+cat)")
var nFilterDirDev = flag.Uint64("filter-dir-dev", 0, "Device ID of filter path parent directory; disambiguates same-inode numbers across different mounts")
var nFilterDirInode2 = flag.Uint64("filter-dir-inode2", 0, "Inode of filter path itself when it is a directory; enables exact-CWD match for relative opens")
var nFilterDirDev2 = flag.Uint64("filter-dir-dev2", 0, "Device ID paired with filter-dir-inode2")
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

	// Populate cgroup filter map after loading (maps are only accessible post-load).
	if *nCgroupPath != "" {
		initCgroupMap(bpfModule)
	}

	// reads data from the trace pipe that bpf_trace_printk() writes to,
	// (/sys/kernel/debug/tracing/trace_pipe).
	go helpers.TracePipeListen()

	// Load the BPF program
	prog, err := bpfModule.GetProgram("injection_disk_failure")
	must(err)

	// Attach the kprobe to the arch-specific openat syscall wrapper.
	// The arch wrapper is tagged ALLOW_ERROR_INJECTION so bpf_override_return works.
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

// initGlobalVariables sets BPF global variables from command-line flags.
func initGlobalVariables(bpfModule *bpf.Module) {
	flag.Parse()

	var pid uint32
	pid = uint32(*nPid)
	if err := bpfModule.InitGlobalVariable("target_pid", pid); err != nil {
		must(err)
	}

	// Enable cgroup-based filtering when a cgroup path is provided.
	var useCgroupFilter int32
	if *nCgroupPath != "" {
		useCgroupFilter = 1
	}
	if err := bpfModule.InitGlobalVariable("use_cgroup_filter", useCgroupFilter); err != nil {
		must(err)
	}

	// Filter directory inode for relative-path disruption.
	filterDirInode := *nFilterDirInode
	if err := bpfModule.InitGlobalVariable("filter_dir_inode", filterDirInode); err != nil {
		must(err)
	}

	// Filter directory device ID — disambiguates same-inode numbers across mounts.
	filterDirDev := uint32(*nFilterDirDev)
	if err := bpfModule.InitGlobalVariable("filter_dir_dev", filterDirDev); err != nil {
		must(err)
	}

	// Second inode/device for exact-CWD matching when filter path is a directory.
	filterDirInode2 := *nFilterDirInode2
	if err := bpfModule.InitGlobalVariable("filter_dir_inode2", filterDirInode2); err != nil {
		must(err)
	}
	filterDirDev2 := uint32(*nFilterDirDev2)
	if err := bpfModule.InitGlobalVariable("filter_dir_dev2", filterDirDev2); err != nil {
		must(err)
	}

	path := []byte(*nPath)
	if err := bpfModule.InitGlobalVariable("filter_path", path); err != nil {
		must(err)
	}

	var exitCode uint32
	exitCode = uint32(*nExitCode)
	if err := bpfModule.InitGlobalVariable("exit_code", exitCode); err != nil {
		must(err)
	}

	var probability uint32
	probability = uint32(*nProbability)
	if err := bpfModule.InitGlobalVariable("probability", probability); err != nil {
		must(err)
	}

	currentPid := uint32(os.Getpid())
	if err := bpfModule.InitGlobalVariable("exclude_pid", currentPid); err != nil {
		must(err)
	}
}

// initCgroupMap opens the cgroupv2 directory and pins its fd into the target_cgroup
// BPF_MAP_TYPE_CGROUP_ARRAY so the eBPF program can use bpf_current_task_under_cgroup().
func initCgroupMap(bpfModule *bpf.Module) {
	fd, err := syscall.Open(*nCgroupPath, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	must(err)
	defer syscall.Close(fd)

	cgroupMap, err := bpfModule.GetMap("target_cgroup")
	must(err)

	key := uint32(0)
	fdUint := uint32(fd)
	must(cgroupMap.Update(unsafe.Pointer(&key), unsafe.Pointer(&fdUint)))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

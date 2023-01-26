package main

import (
	"C"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	bpf "github.com/aquasecurity/libbpfgo"
	"github.com/aquasecurity/libbpfgo/helpers"
	"os"
	"os/signal"
)

var nFlag = flag.Uint64("p", 0, "Process to check")
var nPath = flag.String("f", "/", "Path to filter")

func main() {
	flag.Parse()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	bpfModule, err := bpf.NewModuleFromFile(obj_name)
	must(err)
	defer bpfModule.Close()

	// Get Pid
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

	err = bpfModule.BPFLoadObject()
	must(err)

	go helpers.TracePipeListen()

	prog, err := bpfModule.GetProgram("injection_bpftrace")
	must(err)
	_, err = prog.AttachKprobe(sys_openat)
	must(err)

	e := make(chan []byte, 300)
	p, err := bpfModule.InitPerfBuf("events", e, nil, 1024)
	must(err)

	p.Start()

	counter := make(map[string]int, 350)
	go func() {
		for data := range e {
			ppid := int(binary.LittleEndian.Uint32(data[0:4]))  // Treat first 4 bytes as LittleEndian Uint32
			pid := int(binary.LittleEndian.Uint32(data[4:8]))   // Treat first 4 bytes as LittleEndian Uint32
			tid := int(binary.LittleEndian.Uint32(data[8:12]))  // Treat first 4 bytes as LittleEndian Uint32
			gid := int(binary.LittleEndian.Uint32(data[12:16])) // Treat first 4 bytes as LittleEndian Uint32
			comm := string(bytes.TrimRight(data[16:], "\x00"))  // Remove excess 0's from comm, treat as string
			counter[comm]++
			fmt.Printf("Disrupt Ppid %d, Pid %d, Tid: %d, Gid: %d, Command: %s\n", ppid, pid, tid, gid, comm)
		}
	}()

	<-sig
	p.Stop()
	for comm, n := range counter {
		fmt.Printf("%s: %d\n", comm, n)
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

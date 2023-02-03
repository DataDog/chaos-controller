// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package main

import (
	"C"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"

	bpf "github.com/aquasecurity/libbpfgo"
)
import "os/exec"

var nFlag = flag.Uint64("p", 0, "Process to check")
var nPath = flag.String("f", "/", "Path to filter")

func main() {
	bpfModule, err := bpf.NewModuleFromFile("bpf-xdp-arm64.bpf.o")
	if err != nil {
		log.Printf("ERROR: %s", err.Error())

		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
	defer bpfModule.Close()

	err = bpfModule.BPFLoadObject()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	xdpProg, err := bpfModule.GetProgram("target")
	if xdpProg == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	log.Printf("Loaded program")

	_, err = xdpProg.AttachXDP("lo")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	log.Printf("Attached XDP program to lo interface")

	eventsChannel := make(chan []byte)
	rb, err := bpfModule.InitRingBuf("events", eventsChannel)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	rb.Start()

	numberOfEventsReceived := 0
	go func() {
		_, err := exec.Command("ping", "localhost", "-c 10").Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(-1)
		}
	}()

recvLoop:

	for {
		b := <-eventsChannel
		if binary.LittleEndian.Uint32(b) != 2021 {
			fmt.Fprintf(os.Stderr, "invalid data retrieved\n")
			os.Exit(-1)
		}
		numberOfEventsReceived++
		if numberOfEventsReceived > 5 {
			break recvLoop
		}
	}

	rb.Stop()
	rb.Close()
}

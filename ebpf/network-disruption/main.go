// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build !cgo
// +build !cgo

package main

/*
#cgo LDFLAGS: -lelf -lz
#include <bpf/bpf.h>
#include <stdlib.h>

// bpf_map_get_next_key_wrap wraps the bpf syscall for Go.
// Returns 0 on success, -1 on error (no more keys).
static int bpf_map_get_next_key_wrap(int fd, const void *key, void *next_key) {
    return bpf_map_get_next_key(fd, key, next_key);
}
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"unsafe"

	goflag "flag"

	"github.com/DataDog/chaos-controller/log"
	"github.com/aquasecurity/libbpfgo"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

const (
	DisruptionRulesMapName = "disruption_rules"

	ActionAllow   = 0
	ActionDisrupt = 1
	ActionDrop    = 2
)

var (
	logger *zap.SugaredLogger

	nIP         = flag.String("ip", "", "CIDR to match (e.g., 10.0.0.1/32 or 10.0.0.0/8)")
	nAction     = flag.String("action", "disrupt", "Action: allow, disrupt, or drop")
	nDropPct    = flag.Uint32("drop-pct", 0, "Drop percentage (0-100, for drop action)")
	nIFBIfindex = flag.Uint32("ifb-ifindex", 0, "IFB device ifindex for ingress shaping redirect")
	nSrcPort    = flag.Uint16("src-port", 0, "Source port to match (0 = match all)")
	nDstPort    = flag.Uint16("dst-port", 0, "Destination port to match (0 = match all)")
	nProtocol   = flag.String("protocol", "", "IP protocol to match: tcp, udp (empty = match all)")
	nClear      = flag.Bool("clear", false, "Clear all entries from the disruption rules map")
)

// lpmKey matches struct lpm_key in the BPF program.
// IPv4 addresses are stored as IPv4-mapped IPv6: ::ffff:x.x.x.x
type lpmKey struct {
	PrefixLen uint32
	IP        [16]byte
}

// lpmVal matches struct lpm_val in the BPF program.
// Total size: 20 bytes (3x uint32 + 2x uint16 + 1x uint8 + 3 pad bytes)
type lpmVal struct {
	Action     uint32
	DropPct    uint32
	IFBIfindex uint32
	SrcPort    uint16
	DstPort    uint16
	Protocol   uint8
	Pad        [3]byte
}

func main() {
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	var err error

	logger, err = log.NewZapLogger()
	if err != nil {
		panic(fmt.Sprintf("could not initialize the logger: %v", err))
	}

	if *nClear {
		if err := clearMap(); err != nil {
			logger.Fatalf("could not clear disruption rules map: %v", err)
		}

		logger.Info("disruption rules map cleared")

		return
	}

	if *nIP == "" {
		logger.Fatal("--ip is required")
	}

	if *nDropPct > 100 {
		logger.Fatalf("--drop-pct must be 0-100, got %d", *nDropPct)
	}

	if err := addEntry(); err != nil {
		logger.Fatalf("could not add entry to disruption rules map: %v", err)
	}

	logger.Infof("entry added to disruption rules map: ip=%s action=%s drop_pct=%d ifb_ifindex=%d src_port=%d dst_port=%d protocol=%s",
		*nIP, *nAction, *nDropPct, *nIFBIfindex, *nSrcPort, *nDstPort, *nProtocol)
}

func parseAction(s string) (uint32, error) {
	switch s {
	case "allow":
		return ActionAllow, nil
	case "disrupt":
		return ActionDisrupt, nil
	case "drop":
		return ActionDrop, nil
	default:
		return 0, fmt.Errorf("unknown action: %s", s)
	}
}

// cidrToLPMKey converts a CIDR string to an LPM trie key with IPv4-mapped IPv6 format.
func cidrToLPMKey(cidr string) (lpmKey, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return lpmKey{}, fmt.Errorf("invalid CIDR %s: %w", cidr, err)
	}

	key := lpmKey{}
	ones, _ := ipNet.Mask.Size()

	ip4 := ipNet.IP.To4()
	if ip4 != nil {
		// IPv4: store as ::ffff:x.x.x.x with prefix_len = 96 + ones
		key.IP[10] = 0xff
		key.IP[11] = 0xff
		copy(key.IP[12:16], ip4)
		key.PrefixLen = uint32(96 + ones)
	} else {
		// IPv6: store directly
		ip6 := ipNet.IP.To16()
		if ip6 == nil {
			return lpmKey{}, fmt.Errorf("invalid IP in CIDR %s", cidr)
		}

		copy(key.IP[:], ip6)
		key.PrefixLen = uint32(ones)
	}

	return key, nil
}

func parseProtocol(s string) (uint8, error) {
	switch s {
	case "tcp":
		return 6, nil // IPPROTO_TCP
	case "udp":
		return 17, nil // IPPROTO_UDP
	case "":
		return 0, nil // match all
	default:
		return 0, fmt.Errorf("unknown protocol: %s (supported: tcp, udp)", s)
	}
}

func addEntry() error {
	action, err := parseAction(*nAction)
	if err != nil {
		return err
	}

	key, err := cidrToLPMKey(*nIP)
	if err != nil {
		return err
	}

	proto, err := parseProtocol(*nProtocol)
	if err != nil {
		return err
	}

	val := lpmVal{
		Action:     action,
		DropPct:    *nDropPct,
		IFBIfindex: *nIFBIfindex,
		SrcPort:    *nSrcPort,
		DstPort:    *nDstPort,
		Protocol:   proto,
	}

	bpfMapIDs, err := libbpfgo.GetMapsIDsByName(DisruptionRulesMapName)
	if err != nil {
		return fmt.Errorf("could not get map IDs for %s: %w", DisruptionRulesMapName, err)
	}

	for _, id := range bpfMapIDs {
		bpfMap, err := libbpfgo.GetMapByID(id)
		if err != nil {
			return fmt.Errorf("could not get map with ID %d: %w", id, err)
		}

		keyBytes := make([]byte, unsafe.Sizeof(key))
		binary.LittleEndian.PutUint32(keyBytes[0:4], key.PrefixLen)
		copy(keyBytes[4:20], key.IP[:])

		valBytes := make([]byte, 20) // sizeof(struct lpm_val) = 20 bytes
		binary.LittleEndian.PutUint32(valBytes[0:4], val.Action)
		binary.LittleEndian.PutUint32(valBytes[4:8], val.DropPct)
		binary.LittleEndian.PutUint32(valBytes[8:12], val.IFBIfindex)
		binary.LittleEndian.PutUint16(valBytes[12:14], val.SrcPort)
		binary.LittleEndian.PutUint16(valBytes[14:16], val.DstPort)
		valBytes[16] = val.Protocol
		// bytes 17-19 are padding (zero)

		if err := bpfMap.Update(unsafe.Pointer(&keyBytes[0]), unsafe.Pointer(&valBytes[0])); err != nil {
			closeMap(bpfMap)

			return fmt.Errorf("could not update map entry: %w", err)
		}

		closeMap(bpfMap)
	}

	return nil
}

func clearMap() error {
	bpfMapIDs, err := libbpfgo.GetMapsIDsByName(DisruptionRulesMapName)
	if err != nil {
		return fmt.Errorf("could not get map IDs for %s: %w", DisruptionRulesMapName, err)
	}

	for _, id := range bpfMapIDs {
		bpfMap, err := libbpfgo.GetMapByID(id)
		if err != nil {
			return fmt.Errorf("could not get map with ID %d: %w", id, err)
		}

		// For LPM trie maps, drain by always fetching the first key and deleting it.
		// libbpfgo's GetNextKey is not implemented, so we use the C bpf syscall directly.
		fd := bpfMap.FileDescriptor()
		keySize := 20 // sizeof(lpm_key): 4 (prefix_len) + 16 (ip)
		key := C.malloc(C.ulong(keySize))

		defer C.free(key)

		for {
			// Pass NULL as prev key to get the first remaining entry
			ret := C.bpf_map_get_next_key_wrap(C.int(fd), nil, key)
			if ret != 0 {
				break // No more entries
			}

			if err := bpfMap.DeleteKey(key); err != nil {
				logger.Warnf("could not delete map entry: %v", err)

				break // Avoid infinite loop if delete fails
			}
		}

		closeMap(bpfMap)
	}

	return nil
}

func closeMap(bpfMap *libbpfgo.BPFMapLow) {
	if err := syscall.Close(bpfMap.FileDescriptor()); err != nil {
		logger.Warnf("failed to close map FD %d: %v", bpfMap.FileDescriptor(), err)
	}
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package network

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

// TcMock is a mock implementation of the Tc interface
type TcMock struct {
	mock.Mock
	ListFiltersCallNumber int
}

//nolint:golint
func (f *TcMock) AddNetem(ifaces []string, parent string, handle uint32, delay time.Duration, delayJitter time.Duration, drop int, corrupt int, duplicate int) error {
	args := f.Called(ifaces, parent, handle, delay, delayJitter, drop, corrupt, duplicate)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddPrio(ifaces []string, parent string, handle uint32, bands uint32, priomap [16]uint32) error {
	args := f.Called(ifaces, parent, handle, bands, priomap)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddFilter(ifaces []string, parent string, handle uint32, srcIP, dstIP *net.IPNet, srcPort, dstPort int, protocol string, flowid string) error {
	srcIPs := "nil"
	dstIPs := "nil"

	if srcIP != nil {
		srcIPs = srcIP.String()
	}

	if dstIP != nil {
		dstIPs = dstIP.String()
	}

	log.Printf("%s %s", srcIPs, dstIPs)

	args := f.Called(ifaces, parent, handle, srcIPs, dstIPs, srcPort, dstPort, protocol, flowid)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddCgroupFilter(ifaces []string, parent string, handle uint32) error {
	args := f.Called(ifaces, parent, handle)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddOutputLimit(ifaces []string, parent string, handle uint32, bytesPerSec uint) error {
	args := f.Called(ifaces, parent, handle, bytesPerSec)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) ClearQdisc(ifaces []string) error {
	args := f.Called(ifaces)

	return args.Error(0)
}

// ListFilters is called multiple times in inj.Inject and needs to have different return values
// depending on the number of the call.
func (f *TcMock) ListFilters(ifaces []string) (map[string]string, error) {
	args := f.Called(ifaces)

	if args.Get(f.ListFiltersCallNumber) == nil {
		return nil, fmt.Errorf("argument %d doesn't exist", f.ListFiltersCallNumber)
	}

	arg, ok := args.Get(f.ListFiltersCallNumber).(map[string]string)
	if !ok {
		return nil, fmt.Errorf("argument %d is of the wrong type", f.ListFiltersCallNumber)
	}
	// count the number of calls to return the argument of the number of the call in order to return different values
	f.ListFiltersCallNumber++

	return arg, nil
}

func (f *TcMock) DeleteFilter(iface string, preference string) error {
	args := f.Called(iface, preference)

	return args.Error(0)
}

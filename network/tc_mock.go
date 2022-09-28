// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package network

import (
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

// TcMock is a mock implementation of the Tc interface
type TcMock struct {
	mock.Mock
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
func (f *TcMock) AddFilter(ifaces []string, parent string, priority uint32, handle uint32, srcIP, dstIP *net.IPNet, srcPort, dstPort int, protocol Protocol, connState ConnState, flowid string) error {
	srcIPs := "nil"
	dstIPs := "nil"

	if srcIP != nil {
		srcIPs = srcIP.String()
	}

	if dstIP != nil {
		dstIPs = dstIP.String()
	}

	args := f.Called(ifaces, parent, priority, handle, srcIPs, dstIPs, srcPort, dstPort, string(protocol), string(connState), flowid)

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

func (f *TcMock) DeleteFilter(iface string, priority uint32) error {
	args := f.Called(iface, priority)

	return args.Error(0)
}

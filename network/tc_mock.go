// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import (
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

// TcMock is a mock implementation of the Tc interface
type TcMock struct {
	mock.Mock
	tcPriority uint32
}

func NewTcMock() *TcMock {
	return &TcMock{
		tcPriority: 1000,
	}
}

//nolint:golint
func (f *TcMock) AddNetem(ifaces []string, parent string, handle string, delay time.Duration, delayJitter time.Duration, drop int, corrupt int, duplicate int) error {
	args := f.Called(ifaces, parent, handle, delay, delayJitter, drop, corrupt, duplicate)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddPrio(ifaces []string, parent string, handle string, bands uint32, priomap [16]uint32) error {
	args := f.Called(ifaces, parent, handle, bands, priomap)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddFilter(ifaces []string, parent string, handle string, srcIP, dstIP *net.IPNet, srcPort, dstPort int, protocol Protocol, connState connState, flowid string) (uint32, error) {
	srcIPs := "nil"
	dstIPs := "nil"

	if srcIP != nil {
		srcIPs = srcIP.String()
	}

	if dstIP != nil {
		dstIPs = dstIP.String()
	}

	f.tcPriority++

	args := f.Called(ifaces, parent, handle, srcIPs, dstIPs, srcPort, dstPort, string(protocol), string(connState), flowid)

	return f.tcPriority, args.Error(0)
}

//nolint:golint
func (f *TcMock) AddFwFilter(ifaces []string, parent string, handle string, flowid string) error {
	args := f.Called(ifaces, parent, handle, flowid)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddOutputLimit(ifaces []string, parent string, handle string, bytesPerSec uint) error {
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

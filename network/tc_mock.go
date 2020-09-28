// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

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
func (f *TcMock) AddNetem(iface string, parent string, handle uint32, delay time.Duration, drop int, corrupt int) error {
	args := f.Called(iface, parent, handle, delay, drop, corrupt)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddPrio(iface string, parent string, handle uint32, bands uint32, priomap [16]uint32) error {
	args := f.Called(iface, parent, handle, bands, priomap)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddFilter(iface string, parent string, handle uint32, ip *net.IPNet, port int, protocol string, flowid string, flow string) error {
	ips := "nil"

	if ip != nil {
		ips = ip.String()
	}

	args := f.Called(iface, parent, handle, ips, port, protocol, flowid, flow)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) AddOutputLimit(iface string, parent string, handle uint32, bytesPerSec uint) error {
	args := f.Called(iface, parent, handle, bytesPerSec)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) ClearQdisc(iface string) error {
	args := f.Called(iface)

	return args.Error(0)
}

//nolint:golint
func (f *TcMock) IsQdiscCleared(iface string) (bool, error) {
	args := f.Called(iface)

	return args.Bool(0), args.Error(1)
}

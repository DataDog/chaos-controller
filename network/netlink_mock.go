// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package network

import (
	"net"

	"github.com/stretchr/testify/mock"
)

// NetlinkAdapterMock is a mock implementation of the NetlinkAdapter interface
type NetlinkAdapterMock struct {
	mock.Mock
}

//nolint:golint
func (f *NetlinkAdapterMock) LinkList() ([]NetlinkLink, error) {
	args := f.Called()

	return args.Get(0).([]NetlinkLink), args.Error(1)
}

//nolint:golint
func (f *NetlinkAdapterMock) LinkByIndex(index int) (NetlinkLink, error) {
	args := f.Called(index)

	return args.Get(0).(NetlinkLink), args.Error(1)
}

//nolint:golint
func (f *NetlinkAdapterMock) LinkByName(name string) (NetlinkLink, error) {
	args := f.Called(name)

	return args.Get(0).(NetlinkLink), args.Error(1)
}

//nolint:golint
func (f *NetlinkAdapterMock) DefaultRoute() (NetlinkRoute, error) {
	args := f.Called()

	return args.Get(0).(NetlinkRoute), args.Error(1)
}

// NetlinkLinkMock is a mock implementation of the NetlinkLink interface
type NetlinkLinkMock struct {
	mock.Mock
}

//nolint:golint
func (f *NetlinkLinkMock) Name() string {
	args := f.Called()

	return args.String(0)
}

//nolint:golint
func (f *NetlinkLinkMock) SetTxQLen(qlen int) error {
	args := f.Called(qlen)

	return args.Error(0)
}

//nolint:golint
func (f *NetlinkLinkMock) TxQLen() int {
	args := f.Called()

	return args.Int(0)
}

// NetlinkRouteMock is a mock implementation of the NetlinkRoute interface
type NetlinkRouteMock struct {
	mock.Mock
}

//nolint:golint
func (f *NetlinkRouteMock) Link() NetlinkLink {
	args := f.Called()

	return args.Get(0).(NetlinkLink)
}

//nolint:golint
func (f *NetlinkRouteMock) Gateway() net.IP {
	args := f.Called()

	return args.Get(0).(net.IP)
}

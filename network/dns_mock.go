// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import (
	"net"

	"github.com/stretchr/testify/mock"
)

// DNSMock is the mock implement of the DNSResolver interface
type DNSMock struct {
	mock.Mock
}

//nolint:golint
func (f *DNSMock) Resolve(host string) ([]net.IP, error) {
	args := f.Called(host)

	return args.Get(0).([]net.IP), args.Error(1)
}

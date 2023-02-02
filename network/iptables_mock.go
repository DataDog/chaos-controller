// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package network

import "github.com/stretchr/testify/mock"

// IptablesMock is a mock implementation of the Iptables interface
type IptablesMock struct {
	mock.Mock
}

//nolint:golint
func (f *IptablesMock) CreateChain(name string) error {
	args := f.Called(name)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) ClearAndDeleteChain(name string) error {
	args := f.Called(name)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) AddRuleWithIP(chain string, protocol string, port string, jump string, destinationIP string) error {
	args := f.Called(chain, protocol, port, jump, destinationIP)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) PrependRuleSpec(chain string, rulespec ...string) error {
	args := f.Called(chain, rulespec)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) DeleteRule(chain string, protocol string, port string, jump string) error {
	args := f.Called(chain, protocol, port, jump)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) DeleteRuleSpec(chain string, rulespec ...string) error {
	args := f.Called(chain, rulespec)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) AddWideFilterRule(chain string, protocol string, port string, jump string) error {
	args := f.Called(chain, protocol, port, jump)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) AddCgroupFilterRule(table string, chain string, cgroupPath string, rulespec ...string) error {
	args := f.Called(table, chain, cgroupPath, rulespec)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) DeleteCgroupFilterRule(table string, chain string, cgroupPath string, rulespec ...string) error {
	args := f.Called(table, chain, cgroupPath, rulespec)

	return args.Error(0)
}

//nolint:golint
func (f *IptablesMock) ListRules(table string, chain string) ([]string, error) {
	args := f.Called(table, chain)

	return args.Get(0).([]string), args.Error(1)
}

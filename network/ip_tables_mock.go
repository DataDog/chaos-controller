// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package network

import mock "github.com/stretchr/testify/mock"

// IPTablesMock is an autogenerated mock type for the IPTables type
type IPTablesMock struct {
	mock.Mock
}

type IPTablesMock_Expecter struct {
	mock *mock.Mock
}

func (_m *IPTablesMock) EXPECT() *IPTablesMock_Expecter {
	return &IPTablesMock_Expecter{mock: &_m.Mock}
}

// Clear provides a mock function with given fields:
func (_m *IPTablesMock) Clear() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Clear")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IPTablesMock_Clear_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Clear'
type IPTablesMock_Clear_Call struct {
	*mock.Call
}

// Clear is a helper method to define mock.On call
func (_e *IPTablesMock_Expecter) Clear() *IPTablesMock_Clear_Call {
	return &IPTablesMock_Clear_Call{Call: _e.mock.On("Clear")}
}

func (_c *IPTablesMock_Clear_Call) Run(run func()) *IPTablesMock_Clear_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *IPTablesMock_Clear_Call) Return(_a0 error) *IPTablesMock_Clear_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *IPTablesMock_Clear_Call) RunAndReturn(run func() error) *IPTablesMock_Clear_Call {
	_c.Call.Return(run)
	return _c
}

// Intercept provides a mock function with given fields: protocol, port, cgroupPath, cgroupClassID, injectorPodIP
func (_m *IPTablesMock) Intercept(protocol string, port string, cgroupPath string, cgroupClassID string, injectorPodIP string) error {
	ret := _m.Called(protocol, port, cgroupPath, cgroupClassID, injectorPodIP)

	if len(ret) == 0 {
		panic("no return value specified for Intercept")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, string, string) error); ok {
		r0 = rf(protocol, port, cgroupPath, cgroupClassID, injectorPodIP)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IPTablesMock_Intercept_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Intercept'
type IPTablesMock_Intercept_Call struct {
	*mock.Call
}

// Intercept is a helper method to define mock.On call
//   - protocol string
//   - port string
//   - cgroupPath string
//   - cgroupClassID string
//   - injectorPodIP string
func (_e *IPTablesMock_Expecter) Intercept(protocol interface{}, port interface{}, cgroupPath interface{}, cgroupClassID interface{}, injectorPodIP interface{}) *IPTablesMock_Intercept_Call {
	return &IPTablesMock_Intercept_Call{Call: _e.mock.On("Intercept", protocol, port, cgroupPath, cgroupClassID, injectorPodIP)}
}

func (_c *IPTablesMock_Intercept_Call) Run(run func(protocol string, port string, cgroupPath string, cgroupClassID string, injectorPodIP string)) *IPTablesMock_Intercept_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].(string), args[3].(string), args[4].(string))
	})
	return _c
}

func (_c *IPTablesMock_Intercept_Call) Return(_a0 error) *IPTablesMock_Intercept_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *IPTablesMock_Intercept_Call) RunAndReturn(run func(string, string, string, string, string) error) *IPTablesMock_Intercept_Call {
	_c.Call.Return(run)
	return _c
}

// LogConntrack provides a mock function with given fields:
func (_m *IPTablesMock) LogConntrack() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for LogConntrack")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IPTablesMock_LogConntrack_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'LogConntrack'
type IPTablesMock_LogConntrack_Call struct {
	*mock.Call
}

// LogConntrack is a helper method to define mock.On call
func (_e *IPTablesMock_Expecter) LogConntrack() *IPTablesMock_LogConntrack_Call {
	return &IPTablesMock_LogConntrack_Call{Call: _e.mock.On("LogConntrack")}
}

func (_c *IPTablesMock_LogConntrack_Call) Run(run func()) *IPTablesMock_LogConntrack_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *IPTablesMock_LogConntrack_Call) Return(_a0 error) *IPTablesMock_LogConntrack_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *IPTablesMock_LogConntrack_Call) RunAndReturn(run func() error) *IPTablesMock_LogConntrack_Call {
	_c.Call.Return(run)
	return _c
}

// MarkCgroupPath provides a mock function with given fields: cgroupPath, mark
func (_m *IPTablesMock) MarkCgroupPath(cgroupPath string, mark string) error {
	ret := _m.Called(cgroupPath, mark)

	if len(ret) == 0 {
		panic("no return value specified for MarkCgroupPath")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(cgroupPath, mark)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IPTablesMock_MarkCgroupPath_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MarkCgroupPath'
type IPTablesMock_MarkCgroupPath_Call struct {
	*mock.Call
}

// MarkCgroupPath is a helper method to define mock.On call
//   - cgroupPath string
//   - mark string
func (_e *IPTablesMock_Expecter) MarkCgroupPath(cgroupPath interface{}, mark interface{}) *IPTablesMock_MarkCgroupPath_Call {
	return &IPTablesMock_MarkCgroupPath_Call{Call: _e.mock.On("MarkCgroupPath", cgroupPath, mark)}
}

func (_c *IPTablesMock_MarkCgroupPath_Call) Run(run func(cgroupPath string, mark string)) *IPTablesMock_MarkCgroupPath_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string))
	})
	return _c
}

func (_c *IPTablesMock_MarkCgroupPath_Call) Return(_a0 error) *IPTablesMock_MarkCgroupPath_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *IPTablesMock_MarkCgroupPath_Call) RunAndReturn(run func(string, string) error) *IPTablesMock_MarkCgroupPath_Call {
	_c.Call.Return(run)
	return _c
}

// MarkClassID provides a mock function with given fields: classid, mark
func (_m *IPTablesMock) MarkClassID(classid string, mark string) error {
	ret := _m.Called(classid, mark)

	if len(ret) == 0 {
		panic("no return value specified for MarkClassID")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(classid, mark)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IPTablesMock_MarkClassID_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'MarkClassID'
type IPTablesMock_MarkClassID_Call struct {
	*mock.Call
}

// MarkClassID is a helper method to define mock.On call
//   - classid string
//   - mark string
func (_e *IPTablesMock_Expecter) MarkClassID(classid interface{}, mark interface{}) *IPTablesMock_MarkClassID_Call {
	return &IPTablesMock_MarkClassID_Call{Call: _e.mock.On("MarkClassID", classid, mark)}
}

func (_c *IPTablesMock_MarkClassID_Call) Run(run func(classid string, mark string)) *IPTablesMock_MarkClassID_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string))
	})
	return _c
}

func (_c *IPTablesMock_MarkClassID_Call) Return(_a0 error) *IPTablesMock_MarkClassID_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *IPTablesMock_MarkClassID_Call) RunAndReturn(run func(string, string) error) *IPTablesMock_MarkClassID_Call {
	_c.Call.Return(run)
	return _c
}

// RedirectTo provides a mock function with given fields: protocol, port, destinationIP
func (_m *IPTablesMock) RedirectTo(protocol string, port string, destinationIP string) error {
	ret := _m.Called(protocol, port, destinationIP)

	if len(ret) == 0 {
		panic("no return value specified for RedirectTo")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string) error); ok {
		r0 = rf(protocol, port, destinationIP)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// IPTablesMock_RedirectTo_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RedirectTo'
type IPTablesMock_RedirectTo_Call struct {
	*mock.Call
}

// RedirectTo is a helper method to define mock.On call
//   - protocol string
//   - port string
//   - destinationIP string
func (_e *IPTablesMock_Expecter) RedirectTo(protocol interface{}, port interface{}, destinationIP interface{}) *IPTablesMock_RedirectTo_Call {
	return &IPTablesMock_RedirectTo_Call{Call: _e.mock.On("RedirectTo", protocol, port, destinationIP)}
}

func (_c *IPTablesMock_RedirectTo_Call) Run(run func(protocol string, port string, destinationIP string)) *IPTablesMock_RedirectTo_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(string), args[2].(string))
	})
	return _c
}

func (_c *IPTablesMock_RedirectTo_Call) Return(_a0 error) *IPTablesMock_RedirectTo_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *IPTablesMock_RedirectTo_Call) RunAndReturn(run func(string, string, string) error) *IPTablesMock_RedirectTo_Call {
	_c.Call.Return(run)
	return _c
}

// NewIPTablesMock creates a new instance of IPTablesMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIPTablesMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *IPTablesMock {
	mock := &IPTablesMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

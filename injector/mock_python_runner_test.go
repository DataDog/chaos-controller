// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package injector

import mock "github.com/stretchr/testify/mock"

// MockPythonRunner is an autogenerated mock type for the PythonRunner type
type MockPythonRunner struct {
	mock.Mock
}

type MockPythonRunner_Expecter struct {
	mock *mock.Mock
}

func (_m *MockPythonRunner) EXPECT() *MockPythonRunner_Expecter {
	return &MockPythonRunner_Expecter{mock: &_m.Mock}
}

// RunPython provides a mock function with given fields: args
func (_m *MockPythonRunner) RunPython(args ...string) error {
	_va := make([]interface{}, len(args))
	for _i := range args {
		_va[_i] = args[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...string) error); ok {
		r0 = rf(args...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockPythonRunner_RunPython_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'RunPython'
type MockPythonRunner_RunPython_Call struct {
	*mock.Call
}

// RunPython is a helper method to define mock.On call
//   - args ...string
func (_e *MockPythonRunner_Expecter) RunPython(args ...interface{}) *MockPythonRunner_RunPython_Call {
	return &MockPythonRunner_RunPython_Call{Call: _e.mock.On("RunPython",
		append([]interface{}{}, args...)...)}
}

func (_c *MockPythonRunner_RunPython_Call) Run(run func(args ...string)) *MockPythonRunner_RunPython_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]string, len(args)-0)
		for i, a := range args[0:] {
			if a != nil {
				variadicArgs[i] = a.(string)
			}
		}
		run(variadicArgs...)
	})
	return _c
}

func (_c *MockPythonRunner_RunPython_Call) Return(_a0 error) *MockPythonRunner_RunPython_Call {
	_c.Call.Return(_a0)
	return _c
}

type mockConstructorTestingTNewMockPythonRunner interface {
	mock.TestingT
	Cleanup(func())
}

// NewMockPythonRunner creates a new instance of MockPythonRunner. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockPythonRunner(t mockConstructorTestingTNewMockPythonRunner) *MockPythonRunner {
	mock := &MockPythonRunner{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EventRecorderMock is an autogenerated mock type for the EventRecorder type
type EventRecorderMock struct {
	mock.Mock
}

type EventRecorderMock_Expecter struct {
	mock *mock.Mock
}

func (_m *EventRecorderMock) EXPECT() *EventRecorderMock_Expecter {
	return &EventRecorderMock_Expecter{mock: &_m.Mock}
}

// AnnotatedEventf provides a mock function with given fields: object, annotations, eventtype, reason, messageFmt, args
func (_m *EventRecorderMock) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, object, annotations, eventtype, reason, messageFmt)
	_ca = append(_ca, args...)
	_m.Called(_ca...)
}

// EventRecorderMock_AnnotatedEventf_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'AnnotatedEventf'
type EventRecorderMock_AnnotatedEventf_Call struct {
	*mock.Call
}

// AnnotatedEventf is a helper method to define mock.On call
//   - object runtime.Object
//   - annotations map[string]string
//   - eventtype string
//   - reason string
//   - messageFmt string
//   - args ...interface{}
func (_e *EventRecorderMock_Expecter) AnnotatedEventf(object interface{}, annotations interface{}, eventtype interface{}, reason interface{}, messageFmt interface{}, args ...interface{}) *EventRecorderMock_AnnotatedEventf_Call {
	return &EventRecorderMock_AnnotatedEventf_Call{Call: _e.mock.On("AnnotatedEventf",
		append([]interface{}{object, annotations, eventtype, reason, messageFmt}, args...)...)}
}

func (_c *EventRecorderMock_AnnotatedEventf_Call) Run(run func(object runtime.Object, annotations map[string]string, eventtype string, reason string, messageFmt string, args ...interface{})) *EventRecorderMock_AnnotatedEventf_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-5)
		for i, a := range args[5:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(args[0].(runtime.Object), args[1].(map[string]string), args[2].(string), args[3].(string), args[4].(string), variadicArgs...)
	})
	return _c
}

func (_c *EventRecorderMock_AnnotatedEventf_Call) Return() *EventRecorderMock_AnnotatedEventf_Call {
	_c.Call.Return()
	return _c
}

func (_c *EventRecorderMock_AnnotatedEventf_Call) RunAndReturn(run func(runtime.Object, map[string]string, string, string, string, ...interface{})) *EventRecorderMock_AnnotatedEventf_Call {
	_c.Run(run)
	return _c
}

// Event provides a mock function with given fields: object, eventtype, reason, message
func (_m *EventRecorderMock) Event(object runtime.Object, eventtype string, reason string, message string) {
	_m.Called(object, eventtype, reason, message)
}

// EventRecorderMock_Event_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Event'
type EventRecorderMock_Event_Call struct {
	*mock.Call
}

// Event is a helper method to define mock.On call
//   - object runtime.Object
//   - eventtype string
//   - reason string
//   - message string
func (_e *EventRecorderMock_Expecter) Event(object interface{}, eventtype interface{}, reason interface{}, message interface{}) *EventRecorderMock_Event_Call {
	return &EventRecorderMock_Event_Call{Call: _e.mock.On("Event", object, eventtype, reason, message)}
}

func (_c *EventRecorderMock_Event_Call) Run(run func(object runtime.Object, eventtype string, reason string, message string)) *EventRecorderMock_Event_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(runtime.Object), args[1].(string), args[2].(string), args[3].(string))
	})
	return _c
}

func (_c *EventRecorderMock_Event_Call) Return() *EventRecorderMock_Event_Call {
	_c.Call.Return()
	return _c
}

func (_c *EventRecorderMock_Event_Call) RunAndReturn(run func(runtime.Object, string, string, string)) *EventRecorderMock_Event_Call {
	_c.Run(run)
	return _c
}

// Eventf provides a mock function with given fields: object, eventtype, reason, messageFmt, args
func (_m *EventRecorderMock) Eventf(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{}) {
	var _ca []interface{}
	_ca = append(_ca, object, eventtype, reason, messageFmt)
	_ca = append(_ca, args...)
	_m.Called(_ca...)
}

// EventRecorderMock_Eventf_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Eventf'
type EventRecorderMock_Eventf_Call struct {
	*mock.Call
}

// Eventf is a helper method to define mock.On call
//   - object runtime.Object
//   - eventtype string
//   - reason string
//   - messageFmt string
//   - args ...interface{}
func (_e *EventRecorderMock_Expecter) Eventf(object interface{}, eventtype interface{}, reason interface{}, messageFmt interface{}, args ...interface{}) *EventRecorderMock_Eventf_Call {
	return &EventRecorderMock_Eventf_Call{Call: _e.mock.On("Eventf",
		append([]interface{}{object, eventtype, reason, messageFmt}, args...)...)}
}

func (_c *EventRecorderMock_Eventf_Call) Run(run func(object runtime.Object, eventtype string, reason string, messageFmt string, args ...interface{})) *EventRecorderMock_Eventf_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-4)
		for i, a := range args[4:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(args[0].(runtime.Object), args[1].(string), args[2].(string), args[3].(string), variadicArgs...)
	})
	return _c
}

func (_c *EventRecorderMock_Eventf_Call) Return() *EventRecorderMock_Eventf_Call {
	_c.Call.Return()
	return _c
}

func (_c *EventRecorderMock_Eventf_Call) RunAndReturn(run func(runtime.Object, string, string, string, ...interface{})) *EventRecorderMock_Eventf_Call {
	_c.Run(run)
	return _c
}

// NewEventRecorderMock creates a new instance of EventRecorderMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewEventRecorderMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *EventRecorderMock {
	mock := &EventRecorderMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

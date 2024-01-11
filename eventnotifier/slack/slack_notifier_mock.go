// Code generated by mockery. DO NOT EDIT.

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.
package slack

import (
	slack_goslack "github.com/slack-go/slack"
	mock "github.com/stretchr/testify/mock"
)

// slackNotifierMock is an autogenerated mock type for the slackNotifier type
type slackNotifierMock struct {
	mock.Mock
}

type slackNotifierMock_Expecter struct {
	mock *mock.Mock
}

func (_m *slackNotifierMock) EXPECT() *slackNotifierMock_Expecter {
	return &slackNotifierMock_Expecter{mock: &_m.Mock}
}

// GetUserByEmail provides a mock function with given fields: email
func (_m *slackNotifierMock) GetUserByEmail(email string) (*slack_goslack.User, error) {
	ret := _m.Called(email)

	if len(ret) == 0 {
		panic("no return value specified for GetUserByEmail")
	}

	var r0 *slack_goslack.User
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*slack_goslack.User, error)); ok {
		return rf(email)
	}
	if rf, ok := ret.Get(0).(func(string) *slack_goslack.User); ok {
		r0 = rf(email)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*slack_goslack.User)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// slackNotifierMock_GetUserByEmail_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'GetUserByEmail'
type slackNotifierMock_GetUserByEmail_Call struct {
	*mock.Call
}

// GetUserByEmail is a helper method to define mock.On call
//   - email string
func (_e *slackNotifierMock_Expecter) GetUserByEmail(email interface{}) *slackNotifierMock_GetUserByEmail_Call {
	return &slackNotifierMock_GetUserByEmail_Call{Call: _e.mock.On("GetUserByEmail", email)}
}

func (_c *slackNotifierMock_GetUserByEmail_Call) Run(run func(email string)) *slackNotifierMock_GetUserByEmail_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string))
	})
	return _c
}

func (_c *slackNotifierMock_GetUserByEmail_Call) Return(_a0 *slack_goslack.User, _a1 error) *slackNotifierMock_GetUserByEmail_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *slackNotifierMock_GetUserByEmail_Call) RunAndReturn(run func(string) (*slack_goslack.User, error)) *slackNotifierMock_GetUserByEmail_Call {
	_c.Call.Return(run)
	return _c
}

// PostMessage provides a mock function with given fields: channelID, options
func (_m *slackNotifierMock) PostMessage(channelID string, options ...slack_goslack.MsgOption) (string, string, error) {
	_va := make([]interface{}, len(options))
	for _i := range options {
		_va[_i] = options[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, channelID)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for PostMessage")
	}

	var r0 string
	var r1 string
	var r2 error
	if rf, ok := ret.Get(0).(func(string, ...slack_goslack.MsgOption) (string, string, error)); ok {
		return rf(channelID, options...)
	}
	if rf, ok := ret.Get(0).(func(string, ...slack_goslack.MsgOption) string); ok {
		r0 = rf(channelID, options...)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(string, ...slack_goslack.MsgOption) string); ok {
		r1 = rf(channelID, options...)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(string, ...slack_goslack.MsgOption) error); ok {
		r2 = rf(channelID, options...)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// slackNotifierMock_PostMessage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PostMessage'
type slackNotifierMock_PostMessage_Call struct {
	*mock.Call
}

// PostMessage is a helper method to define mock.On call
//   - channelID string
//   - options ...slack_goslack.MsgOption
func (_e *slackNotifierMock_Expecter) PostMessage(channelID interface{}, options ...interface{}) *slackNotifierMock_PostMessage_Call {
	return &slackNotifierMock_PostMessage_Call{Call: _e.mock.On("PostMessage",
		append([]interface{}{channelID}, options...)...)}
}

func (_c *slackNotifierMock_PostMessage_Call) Run(run func(channelID string, options ...slack_goslack.MsgOption)) *slackNotifierMock_PostMessage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]slack_goslack.MsgOption, len(args)-1)
		for i, a := range args[1:] {
			if a != nil {
				variadicArgs[i] = a.(slack_goslack.MsgOption)
			}
		}
		run(args[0].(string), variadicArgs...)
	})
	return _c
}

func (_c *slackNotifierMock_PostMessage_Call) Return(_a0 string, _a1 string, _a2 error) *slackNotifierMock_PostMessage_Call {
	_c.Call.Return(_a0, _a1, _a2)
	return _c
}

func (_c *slackNotifierMock_PostMessage_Call) RunAndReturn(run func(string, ...slack_goslack.MsgOption) (string, string, error)) *slackNotifierMock_PostMessage_Call {
	_c.Call.Return(run)
	return _c
}

// newSlackNotifierMock creates a new instance of slackNotifierMock. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func newSlackNotifierMock(t interface {
	mock.TestingT
	Cleanup(func())
}) *slackNotifierMock {
	mock := &slackNotifierMock{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

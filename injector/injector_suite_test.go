// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"testing"

	"github.com/DataDog/chaos-fi-controller/container"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

// fake container
type fakeContainer struct {
	mock.Mock
}

func (f *fakeContainer) ID() string {
	return "fake"
}
func (f *fakeContainer) Runtime() container.Runtime {
	return nil
}
func (f *fakeContainer) Netns() container.Netns {
	return nil
}
func (f *fakeContainer) EnterNetworkNamespace() error {
	args := f.Called()
	return args.Error(0)
}
func (f *fakeContainer) ExitNetworkNamespace() error {
	args := f.Called()
	return args.Error(0)
}

var _ = BeforeSuite(func() {
	z, _ := zap.NewDevelopment()
	log = z.Sugar()
})

func TestInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Suite")
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"net"
	"os"
	"testing"

	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger
var ms metrics.MetricsSink

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

// fake dns client
type fakeDNSClient struct {
	mock.Mock
}

func (f *fakeDNSClient) Resolve(host string) ([]net.IP, error) {
	args := f.Called(host)
	return args.Get(0).([]net.IP), args.Error(1)
}

var _ = BeforeSuite(func() {
	z, _ := zap.NewDevelopment()
	log = z.Sugar()
	os.Setenv("STATSD_URL", "localhost:54321")
	ms, _ = metrics.GetSink("noop")
})

var _ = AfterSuite(func() {
	os.Unsetenv("STATSD_URL")
})

func TestInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Suite")
}

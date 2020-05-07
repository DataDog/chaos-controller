// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2020 Datadog, Inc.

package injector_test

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/metrics"
	"github.com/DataDog/chaos-controller/metrics/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"github.com/DataDog/chaos-controller/network"
)

var log *zap.SugaredLogger
var ms metrics.Sink

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

// fake tc
type fakeTc struct {
	mock.Mock
}

func (f* fakeTc) AddCorrupt(iface string, parent string, handle uint32, corrupt int) error{
	args := f.Called(iface,parent,handle,corrupt)
	return args.Error(0)
}

func (f* fakeTc) AddDrop(iface string, parent string, handle uint32, drop int) error{
	args := f.Called(iface,parent,handle,drop)
	return args.Error(0)
}

func (f *fakeTc) AddDelay(iface string, parent string, handle uint32, delay time.Duration) error {
	args := f.Called(iface, parent, handle, delay)
	return args.Error(0)
}

func (f *fakeTc) AddPrio(iface string, parent string, handle uint32, bands uint32, priomap [16]uint32) error {
	args := f.Called(iface, parent, handle, bands, priomap)
	return args.Error(0)
}
func (f *fakeTc) AddFilter(iface string, parent string, handle uint32, ip *net.IPNet, port int, flowid string) error {
	args := f.Called(iface, parent, handle, ip.String(), port, flowid)
	return args.Error(0)
}
func (f *fakeTc) ClearQdisc(iface string) error {
	args := f.Called(iface)
	return args.Error(0)
}
func (f *fakeTc) IsQdiscCleared(iface string) (bool, error) {
	args := f.Called(iface)
	return args.Bool(0), args.Error(1)
}

// netlink
type fakeNetlinkAdapter struct {
	mock.Mock
}

func (f *fakeNetlinkAdapter) LinkList() ([]network.NetlinkLink, error) {
	args := f.Called()
	return args.Get(0).([]network.NetlinkLink), args.Error(1)
}
func (f *fakeNetlinkAdapter) LinkByIndex(index int) (network.NetlinkLink, error) {
	args := f.Called(index)
	return args.Get(0).(network.NetlinkLink), args.Error(1)
}
func (f *fakeNetlinkAdapter) LinkByName(name string) (network.NetlinkLink, error) {
	args := f.Called(name)
	return args.Get(0).(network.NetlinkLink), args.Error(1)
}
func (f *fakeNetlinkAdapter) RoutesForIP(ip *net.IPNet) ([]network.NetlinkRoute, error) {
	args := f.Called(ip.String())
	return args.Get(0).([]network.NetlinkRoute), args.Error(1)
}

// fake dns client
type fakeDNSClient struct {
	mock.Mock
}

type fakeNetlinkLink struct {
	mock.Mock
}

func (f *fakeNetlinkLink) Name() string {
	args := f.Called()
	return args.String(0)
}
func (f *fakeNetlinkLink) SetTxQLen(qlen int) error {
	args := f.Called(qlen)
	return args.Error(0)
}
func (f *fakeNetlinkLink) TxQLen() int {
	args := f.Called()
	return args.Int(0)
}

type fakeNetlinkRoute struct {
	mock.Mock
}

func (f *fakeNetlinkRoute) Link() network.NetlinkLink {
	args := f.Called()
	return args.Get(0).(network.NetlinkLink)
}

func (f *fakeDNSClient) Resolve(host string) ([]net.IP, error) {
	args := f.Called(host)
	return args.Get(0).([]net.IP), args.Error(1)
}

var _ = BeforeSuite(func() {
	z, _ := zap.NewDevelopment()
	log = z.Sugar()
	os.Setenv("STATSD_URL", "localhost:54321")
	ms, _ = metrics.GetSink(types.SinkDriverNoop, types.SinkAppInjector)
})

var _ = AfterSuite(func() {
	os.Unsetenv("STATSD_URL")
})

func TestInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Suite")
}

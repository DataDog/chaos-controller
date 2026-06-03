// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build integration

package injector_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/cgroup"
	ctn "github.com/DataDog/chaos-controller/container"
	. "github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	chaostypes "github.com/DataDog/chaos-controller/types"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	kubernetes "k8s.io/client-go/kubernetes/fake"
)

// bpfCmdRunnerIntegrationMock satisfies bpfdisrupt.CmdRunner; BPF path not exercised.
type bpfCmdRunnerIntegrationMock struct {
	mock.Mock
}

func (m *bpfCmdRunnerIntegrationMock) Run(args []string) (int, string, error) {
	return 0, "", nil
}

// startIsolatedNetwork creates a dedicated Docker bridge network for one test.
// Returns the network name and a cleanup func.
func startIsolatedNetwork(ctx context.Context) (string, func()) {
	net, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: testcontainers.NetworkRequest{
			Name:           fmt.Sprintf("chaos-test-%d", time.Now().UnixNano()),
			Driver:         "bridge",
			CheckDuplicate: true,
		},
	})
	Expect(err).NotTo(HaveOccurred(), "create isolated Docker network")

	dockerNet, ok := net.(*testcontainers.DockerNetwork)
	Expect(ok).To(BeTrue(), "expected *testcontainers.DockerNetwork from GenericNetwork")

	return dockerNet.Name, func() { _ = net.Remove(ctx) }
}

// startTargetContainer starts nginx:alpine on the given network.
// Returns the container and its bridge IP within that network.
func startTargetContainer(ctx context.Context, netName string) (testcontainers.Container, string) {
	req := testcontainers.ContainerRequest{
		Image:    "nginx:alpine",
		Networks: []string{netName},
		WaitingFor: wait.NewHTTPStrategy("/").
			WithPort("80/tcp").
			WithStartupTimeout(30 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred(), "start target container")

	ip, err := c.ContainerIP(ctx)
	Expect(err).NotTo(HaveOccurred(), "get target container IP")

	return c, ip
}

// startSenderContainer starts an alpine container with wget on the given network.
func startSenderContainer(ctx context.Context, netName string) testcontainers.Container {
	req := testcontainers.ContainerRequest{
		Image:    "alpine:latest",
		Networks: []string{netName},
		Cmd:      []string{"sh", "-c", "sleep 3600"},
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred(), "start sender container")

	code, _, err := c.Exec(ctx, []string{"apk", "add", "--quiet", "wget"})
	Expect(err).NotTo(HaveOccurred())
	Expect(code).To(BeZero(), "apk add wget in sender")

	return c
}

// buildNetworkInjector constructs a NetworkDisruptionInjector wired with real
// tc/iptables/netlink/netns drivers and mocked cgroup + BPF. Returns the
// injector and the target container's host PID (needed for netns assertions).
func buildNetworkInjector(ctx context.Context, spec v1beta1.NetworkDisruptionSpec, targetCtnID string) (Injector, uint32) {
	mountProc := os.Getenv("CHAOS_INJECTOR_MOUNT_PROC")
	Expect(mountProc).NotTo(BeEmpty())

	// injector requires node IP env var to build tc safeguard rules
	if os.Getenv("TARGET_POD_HOST_IP") == "" {
		Expect(os.Setenv("TARGET_POD_HOST_IP", "127.0.0.1")).To(Succeed())
	}

	targetCtn, err := ctn.New("docker://"+targetCtnID, "integration-target")
	Expect(err).NotTo(HaveOccurred(), "create container handle for target")

	pid := targetCtn.PID()
	Expect(pid).NotTo(BeZero(), "target container PID must be non-zero")

	netnsMgr, err := netns.NewManager(integrationLog, pid)
	Expect(err).NotTo(HaveOccurred(), "create netns manager")

	// mock cgroup: cgroup v2 path makes all network disruption cgroup writes no-ops
	cgroupMgr := cgroup.NewManagerMock(ginkgo.GinkgoT())
	cgroupMgr.EXPECT().IsCgroupV2().Return(true).Maybe()
	cgroupMgr.EXPECT().RelativePath(mock.Anything).Return("").Maybe()

	tc := network.NewTrafficController(integrationLog, false)
	ipt, err := network.NewIPTables(integrationLog, false)
	Expect(err).NotTo(HaveOccurred(), "create iptables")
	nl := network.NewNetlinkAdapter()

	k8s := kubernetes.NewSimpleClientset()

	disruptionArgs := api.DisruptionArgs{
		Level:          chaostypes.DisruptionLevelPod,
		TargetNodeName: "integration-node",
		DryRun:         false,
	}

	config := NetworkDisruptionInjectorConfig{
		Config: Config{
			Log:         integrationLog,
			MetricsSink: integrationMS,
			Cgroup:      cgroupMgr,
			Netns:       netnsMgr,
			K8sClient:   k8s,
			Disruption:  disruptionArgs,
			InjectorCtx: ctx,
		},
		TrafficController:   tc,
		IPTables:            ipt,
		NetlinkAdapter:      nl,
		BPFDisruptCmdRunner: &bpfCmdRunnerIntegrationMock{},
	}

	inj, err := NewNetworkDisruptionInjector(spec, config)
	Expect(err).NotTo(HaveOccurred(), "create network disruption injector")

	return inj, pid
}

// containerID returns the container ID from a testcontainers.Container.
func containerID(c testcontainers.Container) string {
	return c.GetContainerID()
}

// tcQdiscShow runs `tc qdisc show dev eth0` in the target container's netns
// using nsenter from the test container (which has tc installed). Returns output.
func tcQdiscShow(targetPID uint32) string {
	nsPath := fmt.Sprintf("/proc/%d/ns/net", targetPID)
	out, err := exec.Command("nsenter", "--net="+nsPath, "tc", "qdisc", "show", "dev", "eth0").CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "nsenter tc qdisc show: %s", string(out))
	return string(out)
}

// assertTCQdisc asserts output of tc qdisc show in target netns contains substr.
func assertTCQdisc(targetPID uint32, substr string) {
	out := tcQdiscShow(targetPID)
	Expect(out).To(ContainSubstring(substr),
		"tc qdisc show dev eth0 in netns of pid %d: %q", targetPID, out)
}

// assertTCQdiscAbsent asserts output of tc qdisc show does NOT contain substr.
func assertTCQdiscAbsent(targetPID uint32, substr string) {
	out := tcQdiscShow(targetPID)
	Expect(out).NotTo(ContainSubstring(substr),
		"tc qdisc show dev eth0 in netns of pid %d: %q", targetPID, out)
}

// latencySpec returns a NetworkDisruptionSpec for delay-only disruption.
func latencySpec(delayMs uint) v1beta1.NetworkDisruptionSpec {
	return v1beta1.NetworkDisruptionSpec{
		Delay: delayMs,
	}
}

// splitLines returns non-empty lines from s.
func splitLines(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

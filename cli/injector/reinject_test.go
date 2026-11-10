// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package main

import (
	"testing"

	"github.com/DataDog/chaos-controller/cgroup"
	"github.com/DataDog/chaos-controller/container"
	"github.com/DataDog/chaos-controller/injector"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

func containerMock(t *testing.T, id string) *container.ContainerMock {
	c := container.NewContainerMock(t)
	c.EXPECT().ID().Return(id).Maybe()
	c.EXPECT().Name().Return("app").Maybe()
	c.EXPECT().PID().Return(42).Maybe()

	return c
}

// After a target container restarts, reinjection must record the new
// container so the next pod update is not detected as another restart.
func TestRefreshConfigsRecordsRestartedContainer(t *testing.T) {
	log = zap.NewNop().Sugar()
	restarted := containerMock(t, "restarted")
	injectorMock := injector.NewInjectorMock(t)
	injectorMock.EXPECT().UpdateConfig(mock.Anything).Once()

	configs = []injector.Config{{TargetContainer: containerMock(t, "original")}}
	injectors = []injector.Injector{injectorMock}
	disruptionArgs.TargetContainers = map[string]string{"app": "containerd://restarted"}

	err := refreshConfigs(
		func(id, name string) (container.Container, error) { return restarted, nil },
		func(uint32) (netns.Manager, cgroup.Manager, error) { return nil, nil, nil },
	)
	if err != nil {
		t.Fatal(err)
	}

	if got := configs[0].TargetContainer.ID(); got != "restarted" {
		t.Fatalf("configs still name container %q after reinjection, want %q", got, "restarted")
	}

	pod := corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{Name: "app", ContainerID: "containerd://restarted"}}}}

	changed, err := updateTargetContainersAndDetectChange(pod)
	if err != nil {
		t.Fatal(err)
	}

	if changed {
		t.Fatal("an unchanged pod was detected as a container restart after reinjection")
	}
}

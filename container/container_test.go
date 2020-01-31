package container_test

import (
	"context"
	"reflect"
	"runtime"
	"syscall"

	"bou.ke/monkey"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/types"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	prototypes "github.com/gogo/protobuf/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/vishvananda/netns"

	. "github.com/DataDog/chaos-fi-controller/container"
)

// fakeContainer implements the Container interface
type fakeContainer struct{}

func (fc fakeContainer) ID() string {
	return "fake"
}
func (fc fakeContainer) Info(context.Context, ...containerd.InfoOpts) (containers.Container, error) {
	return containers.Container{}, nil
}
func (fc fakeContainer) Delete(context.Context, ...containerd.DeleteOpts) error { return nil }
func (fc fakeContainer) NewTask(context.Context, cio.Creator, ...containerd.NewTaskOpts) (containerd.Task, error) {
	return nil, nil
}
func (fc fakeContainer) Spec(context.Context) (*oci.Spec, error) { return nil, nil }
func (fc fakeContainer) Task(context.Context, cio.Attach) (containerd.Task, error) {
	return fakeTask{}, nil
}
func (fc fakeContainer) Image(context.Context) (containerd.Image, error)   { return nil, nil }
func (fc fakeContainer) Labels(context.Context) (map[string]string, error) { return nil, nil }
func (fc fakeContainer) SetLabels(context.Context, map[string]string) (map[string]string, error) {
	return nil, nil
}
func (fc fakeContainer) Extensions(context.Context) (map[string]prototypes.Any, error) {
	return nil, nil
}
func (fc fakeContainer) Update(context.Context, ...containerd.UpdateContainerOpts) error { return nil }
func (fc fakeContainer) Checkpoint(context.Context, string, ...containerd.CheckpointOpts) (containerd.Image, error) {
	return nil, nil
}

// fakeTask implement the Task interface
type fakeTask struct{}

func (ft fakeTask) ID() string                  { return "" }
func (ft fakeTask) Pid() uint32                 { return 666 }
func (ft fakeTask) Start(context.Context) error { return nil }
func (ft fakeTask) Delete(context.Context, ...containerd.ProcessDeleteOpts) (*containerd.ExitStatus, error) {
	return nil, nil
}
func (ft fakeTask) Kill(context.Context, syscall.Signal, ...containerd.KillOpts) error { return nil }
func (ft fakeTask) Wait(context.Context) (<-chan containerd.ExitStatus, error)         { return nil, nil }
func (ft fakeTask) CloseIO(context.Context, ...containerd.IOCloserOpts) error          { return nil }
func (ft fakeTask) Resize(ctx context.Context, w, h uint32) error                      { return nil }
func (ft fakeTask) IO() cio.IO                                                         { return nil }
func (ft fakeTask) Status(context.Context) (containerd.Status, error)                  { return containerd.Status{}, nil }
func (ft fakeTask) Pause(context.Context) error                                        { return nil }
func (ft fakeTask) Resume(context.Context) error                                       { return nil }
func (ft fakeTask) Exec(context.Context, string, *specs.Process, cio.Creator) (containerd.Process, error) {
	return nil, nil
}
func (ft fakeTask) Pids(context.Context) ([]containerd.ProcessInfo, error) { return nil, nil }
func (ft fakeTask) Checkpoint(context.Context, ...containerd.CheckpointTaskOpts) (containerd.Image, error) {
	return nil, nil
}
func (ft fakeTask) Update(context.Context, ...containerd.UpdateTaskOpts) error { return nil }
func (ft fakeTask) LoadProcess(context.Context, string, cio.Attach) (containerd.Process, error) {
	return nil, nil
}
func (ft fakeTask) Metrics(context.Context) (*types.Metric, error) { return nil, nil }

// tests
var _ = Describe("Container", func() {
	var fakeRootNamespace, fakeContainerNamespace netns.NsHandle
	var callLockOSThread, callUnlockOSThread bool
	var callNetnsSetArg netns.NsHandle

	BeforeEach(func() {
		// initialize tests variables
		fakeRootNamespace = netns.NsHandle(-1)
		fakeContainerNamespace = netns.NsHandle(-2)
		callLockOSThread = false
		callUnlockOSThread = false
		callNetnsSetArg = netns.NsHandle(0)

		// netns
		monkey.Patch(netns.Get, func() (netns.NsHandle, error) {
			return fakeRootNamespace, nil
		})
		monkey.Patch(netns.GetFromPath, func(string) (netns.NsHandle, error) {
			return fakeContainerNamespace, nil
		})
		monkey.Patch(netns.Set, func(ns netns.NsHandle) error {
			callNetnsSetArg = ns
			return nil
		})

		// containerd
		var c *containerd.Client
		monkey.Patch(containerd.New, func(address string, opts ...containerd.ClientOpt) (*containerd.Client, error) {
			return &containerd.Client{}, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(c), "LoadContainer", func(c *containerd.Client, ctx context.Context, id string) (containerd.Container, error) {
			return fakeContainer{}, nil
		})

		// runtime
		monkey.Patch(runtime.LockOSThread, func() {
			callLockOSThread = true
		})
		monkey.Patch(runtime.UnlockOSThread, func() {
			callUnlockOSThread = true
		})
	})

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("loading a container", func() {
		It("should return a container object with parsed PID and ns handler", func() {
			c, err := New("fake")
			Expect(err).To(BeNil())
			Expect(c.ID).To(Equal("fake"))
			Expect(c.PID).To(Equal(uint32(666)))
			Expect(c.NetworkNamespace).To(Equal(fakeContainerNamespace))
		})
	})

	Describe("entering and exiting the container network namespace", func() {
		It("should lock thread and set the container network namespace", func() {
			c, err := New("fake")
			Expect(err).To(BeNil())

			err = c.EnterNetworkNamespace()
			Expect(err).To(BeNil())

			Expect(callLockOSThread).To(Equal(true))
			Expect(callNetnsSetArg).To(Equal(fakeContainerNamespace))
		})

		It("should unlock thread and set the root network namespace back", func() {
			err := ExitNetworkNamespace()
			Expect(err).To(BeNil())
			Expect(callUnlockOSThread).To(Equal(true))
			Expect(callNetnsSetArg).To(Equal(fakeRootNamespace))
		})
	})
})

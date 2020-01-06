package container

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/DataDog/chaos-fi-controller/pkg/logger"
	"github.com/containerd/containerd"
	"github.com/vishvananda/netns"
)

var instance *containerd.Client
var once sync.Once
var rootNetworkNamespace netns.NsHandle

// Container describes a container
type Container struct {
	ID               string
	PID              uint32
	NetworkNamespace netns.NsHandle
}

// New creates a new container from the given container ID, retrieving it's main PID and network namespace
func New(containerID string) Container {
	return Container{
		ID:               containerID,
		PID:              getPID(containerID),
		NetworkNamespace: getNetworkNamespace(containerID),
	}
}

// getInstance returns an initialized instance of the containerd client using a singleton pattern
func getInstance() *containerd.Client {
	once.Do(func() {
		var err error
		instance, err = containerd.New("/run/containerd/containerd.sock", containerd.WithDefaultNamespace("k8s.io"))
		if err != nil {
			logger.Instance().Fatalw("unable to connect to the containerd socket", "error", err)
		}
	})
	return instance
}

// getPID loads the given container and returns its task PID
func getPID(containerID string) uint32 {
	container, err := getInstance().LoadContainer(context.Background(), containerID)
	if err != nil {
		logger.Instance().Fatalw("error while loading the given container", "error", err, "containerID", containerID)
	}
	task, err := container.Task(context.Background(), nil)
	if err != nil {
		logger.Instance().Fatalw("error while loading the given container task", "error", err, "containerID", containerID)
	}
	return task.Pid()
}

// getNetworkNamespace gets the given container network namespace file from its task PID
func getNetworkNamespace(containerID string) netns.NsHandle {
	pid := getPID(containerID)
	ns, err := netns.GetFromPath(fmt.Sprintf("/mnt/proc/%d/ns/net", pid))
	if err != nil {
		logger.Instance().Fatalw(
			"error while retrieving the given container network namespace",
			"error", err,
			"containerID", containerID,
			"pid", pid,
		)
	}
	return ns
}

// ExitNetworkNamespace returns into the root network namespace
func ExitNetworkNamespace() {
	err := netns.Set(rootNetworkNamespace)
	if err != nil {
		logger.Instance().Fatalw("error while re-entering the root network namespace", "error", err)
	}
	runtime.UnlockOSThread()
}

// EnterNetworkNamespace saves the actual namespace and enters the given container network namespace
func (c Container) EnterNetworkNamespace() {
	var err error
	runtime.LockOSThread()
	rootNetworkNamespace, err = netns.Get()
	if err != nil {
		logger.Instance().Fatalw("error while saving the root network namespace", "error", err)
	}
	err = netns.Set(c.NetworkNamespace)
	if err != nil {
		logger.Instance().Fatalw("error while entering the container network namespace",
			"error", err,
			"containerID", c.ID,
		)
	}
}

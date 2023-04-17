// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package process

import "runtime"

type Runtime interface {
	// GOMAXPROCS sets the maximum number of CPUs that can be executing
	// simultaneously and returns the previous setting. It defaults to
	// the value of runtime.NumCPU. If n < 1, it does not change the current setting.
	// This call will go away when the scheduler improves.	GOMAXPROCS(int) int
	GOMAXPROCS(int) int

	// LockOSThread wires the calling goroutine to its current operating system thread.
	// The calling goroutine will always execute in that thread,
	// and no other goroutine will execute in it,
	// until the calling goroutine has made as many calls to
	// UnlockOSThread as to LockOSThread.
	// If the calling goroutine exits without unlocking the thread,
	// the thread will be terminated.
	//
	// All init functions are run on the startup thread. Calling LockOSThread
	// from an init function will cause the main function to be invoked on
	// that thread.
	//
	// A goroutine should call LockOSThread before calling OS services or
	// non-Go library functions that depend on per-thread state.
	LockOSThread()

	// UnlockOSThread undoes an earlier call to LockOSThread.
	// If this drops the number of active LockOSThread calls on the
	// calling goroutine to zero, it unwires the calling goroutine from
	// its fixed operating system thread.
	// If there are no active LockOSThread calls, this is a no-op.
	//
	// Before calling UnlockOSThread, the caller must ensure that the OS
	// thread is suitable for running other goroutines. If the caller made
	// any permanent changes to the state of the thread that would affect
	// other goroutines, it should not call this function and thus leave
	// the goroutine locked to the OS thread until the goroutine (and
	// hence the thread) exits.
	UnlockOSThread()
}

// runtimeImpl implement Runtime interface and enable to perform dryRun AND testing
type runtimeImpl struct {
	dryRun bool
}

func NewRuntime(dryRun bool) Runtime {
	return &runtimeImpl{
		dryRun: dryRun,
	}
}

func (r runtimeImpl) GOMAXPROCS(new int) int {
	if r.dryRun {
		return new
	}
	return runtime.GOMAXPROCS(new)
}

func (r runtimeImpl) LockOSThread() {
	if r.dryRun {
		return
	}
	runtime.LockOSThread()
}

func (r runtimeImpl) UnlockOSThread() {
	if r.dryRun {
		return
	}
	runtime.UnlockOSThread()
}

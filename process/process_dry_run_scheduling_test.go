// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux

package process_test

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/DataDog/chaos-controller/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"
)

var _ = Describe("Dry-run process scheduling", func() {
	It("skips affinity writes in dry-run mode", func() {
		// An empty affinity mask fails with EINVAL if the syscall is executed,
		// without changing the test runner's existing affinity.
		Expect(process.NewManager(true).SetAffinity(nil)).To(Succeed())
	})

	It("preserves process-group priority during a dry run", func() {
		const helperEnv = "CHAOS_DRY_RUN_PRIORITY_HELPER"
		if os.Getenv(helperEnv) == "1" {
			// This runs only in a subprocess with its own process group.
			// Lower its priority first so a real change to -20 is observable.
			pgid := unix.Getpgrp()
			Expect(unix.Setpriority(unix.PRIO_PGRP, pgid, 19)).To(Succeed())
			before, err := unix.Getpriority(unix.PRIO_PGRP, pgid)
			Expect(err).NotTo(HaveOccurred())
			priorityErr := process.NewManager(true).Prioritize()
			after, err := unix.Getpriority(unix.PRIO_PGRP, pgid)
			Expect(err).NotTo(HaveOccurred())
			Expect(priorityErr).NotTo(HaveOccurred(), "dry-run must not attempt a privileged priority change")
			Expect(after).To(Equal(before), "dry-run changed the process group's priority")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcess$", "-ginkgo.focus=preserves process-group priority during a dry run")
		child.Env = append(os.Environ(), helperEnv+"=1")
		child.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		output, err := child.CombinedOutput()
		Expect(err).NotTo(HaveOccurred(), "isolated priority regression failed:\n%s", output)
	})
})

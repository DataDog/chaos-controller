// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"os"
	"path/filepath"

	goiptables "github.com/coreos/go-iptables/iptables"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = DescribeTable("iptables cleanup releases rule ownership",
	func(partialFailure, removeBeforeClear bool) {
		dir := GinkgoT().TempDir()
		stateDir := filepath.Join(dir, "rules")
		Expect(os.Mkdir(stateDir, 0700)).To(Succeed())
		GinkgoT().Setenv("CHAOS_IPTABLES_TEST_STATE", stateDir)
		executable := filepath.Join(dir, "iptables")
		// Model only the cgroup mark rules exercised here; never invoke real iptables.
		script := `#!/bin/sh
if [ "$1" = "--version" ]; then
    printf 'iptables v1.8.10 (nf_tables)\n'
    exit 0
fi
if [ "$3" = "-X" ]; then
    exit 0
fi
previous=
classid=
for argument do
    if [ "$previous" = "--cgroup" ]; then classid=$argument; fi
    previous=$argument
done
[ -n "$classid" ] || exit 2
state="$CHAOS_IPTABLES_TEST_STATE/$classid"
case "$3" in
    -C) [ -f "$state" ]; exit $? ;;
    -I) : > "$state"; exit 0 ;;
    -D)
        if [ "$classid" = "0x100002" ] && [ -f "$CHAOS_IPTABLES_TEST_STATE/fail-delete" ]; then
            exit 2
        fi
        [ -f "$state" ] || exit 1
        rm "$state"
        exit $?
        ;;
    *) exit 2 ;;
esac
`
		Expect(os.WriteFile(executable, []byte(script), 0755)).To(Succeed())
		backend, err := goiptables.New(goiptables.Path(executable))
		Expect(err).ToNot(HaveOccurred())
		owner := &iptables{log: zap.NewNop().Sugar(), ip: backend}
		other := &iptables{log: zap.NewNop().Sugar(), ip: backend}

		Expect(owner.MarkClassID("0x100001", "0x1")).To(Succeed())
		if removeBeforeClear {
			Expect(os.Remove(filepath.Join(stateDir, "0x100001"))).To(Succeed())
		}
		if partialFailure {
			Expect(owner.MarkClassID("0x100002", "0x2")).To(Succeed())
			Expect(os.WriteFile(filepath.Join(stateDir, "fail-delete"), nil, 0600)).To(Succeed())
			Expect(owner.Clear()).To(HaveOccurred())
		} else {
			Expect(owner.Clear()).To(Succeed())
		}

		// A later injector owns the replacement rule, not the original injector.
		Expect(other.MarkClassID("0x100001", "0x1")).To(Succeed())
		if partialFailure {
			Expect(os.Remove(filepath.Join(stateDir, "fail-delete"))).To(Succeed())
		}
		Expect(owner.Clear()).To(Succeed())
		exists, err := backend.Exists("mangle", "POSTROUTING", "-m", "cgroup", "--cgroup", "0x100001", "-j", "MARK", "--set-mark", "0x1")
		Expect(err).ToNot(HaveOccurred())
		Expect(exists).To(BeTrue(), "a cleanup retry must not remove a later injector's replacement rule")
	},
	Entry("after successful cleanup", false, false),
	Entry("after partial cleanup", true, false),
	Entry("after confirming the original rule was already absent", false, true),
)

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

//go:build linux

package network

import (
	"os"
	"testing"
)

// Listing routes opens netlink sockets; every call must release them.
func TestListRoutesReleasesNetlinkSockets(t *testing.T) {
	adapter := netlinkAdapter{}

	if _, err := adapter.listRoutes(); err != nil {
		t.Fatalf("listing routes: %v", err)
	}

	before := openFDs(t)

	for range 10 {
		if _, err := adapter.listRoutes(); err != nil {
			t.Fatalf("listing routes: %v", err)
		}
	}

	if after := openFDs(t); after != before {
		t.Fatalf("open file descriptors went from %d to %d over 10 route listings", before, after)
	}
}

func openFDs(t *testing.T) int {
	t.Helper()

	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}

	return len(entries)
}

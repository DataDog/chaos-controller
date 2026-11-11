// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cloudservice

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/DataDog/chaos-controller/cloudservice/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cloud refresh HTTP status", func() {
	It("retains the last GCP ranges when a refresh returns a JSON error response", func(ctx SpecContext) {
		var unavailable atomic.Bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if unavailable.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = fmt.Fprint(w, `{"message":"temporarily unavailable"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"syncToken":"1","prefixes":[{"ipv4Prefix":"192.0.2.0/24"}]}`)
		}))
		DeferCleanup(server.Close)

		manager, err := New(ctx, types.CloudProviderConfigs{
			GCP: types.CloudProviderConfig{Enabled: true, IPRangesURL: server.URL},
		}, server.Client())
		Expect(err).NotTo(HaveOccurred())

		unavailable.Store(true)
		refreshErr := manager.PullIPRanges(ctx)
		ranges, err := manager.GetServicesIPRanges(ctx, types.CloudProviderGCP, []string{"Google"})
		Expect(err).NotTo(HaveOccurred())
		Expect(ranges["Google"]).To(Equal([]string{"192.0.2.0/24"}), "a failed HTTP refresh must not erase the last successful ranges")
		Expect(refreshErr).To(HaveOccurred())
		Expect(refreshErr.Error()).To(ContainSubstring("503"))
	})
})

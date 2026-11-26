// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/eventnotifier/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTTP header case-insensitive precedence", func() {
	It("replaces the file header rather than sending both authorization values", func() {
		path := filepath.Join(GinkgoT().TempDir(), "headers")
		Expect(os.WriteFile(path, []byte("Authorization:Bearer old\n"), 0o600)).To(Succeed())
		received := make(chan []string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			received <- r.Header.Values("Authorization")
			w.WriteHeader(http.StatusNoContent)
		}))
		DeferCleanup(server.Close)
		notifier, err := New(types.NotifiersCommonConfig{}, Config{
			HeadersFilepath: path,
			Headers:         []string{"authorization:Bearer new"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(notifier.emitEvent(context.Background(), server.URL, []byte(`{}`))).To(Succeed())
		Expect(<-received).To(Equal([]string{"Bearer new"}))
	})
})

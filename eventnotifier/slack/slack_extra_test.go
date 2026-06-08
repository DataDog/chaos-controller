// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package slack

import (
	"context"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/DataDog/chaos-controller/eventnotifier/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Slack New", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "slack-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, tmpDir)
	})

	It("returns error when token file does not exist", func() {
		_, err := New(context.TODO(), types.NotifiersCommonConfig{}, NotifierSlackConfig{
			TokenFilepath: "/nonexistent/slack-token",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("slack token file notifier found"))
	})

	It("returns error when token file is empty", func() {
		path := filepath.Join(tmpDir, "token")
		Expect(os.WriteFile(path, []byte(""), 0o600)).To(Succeed())
		_, err := New(context.TODO(), types.NotifiersCommonConfig{}, NotifierSlackConfig{
			TokenFilepath: path,
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("seemingly empty"))
	})
})

var _ = Describe("Slack GetNotifierName", func() {
	It("returns slack driver name", func() {
		n := &Notifier{}
		Expect(n.GetNotifierName()).To(Equal(string(types.NotifierDriverSlack)))
	})
})

var _ = Describe("Slack Notify unrecognized object", func() {
	It("returns nil for non-Disruption/DisruptionCron objects", func(ctx SpecContext) {
		n := &Notifier{
			config: NotifierSlackConfig{Enabled: true},
		}
		err := n.Notify(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod"}}, corev1.Event{}, types.NotificationInfo)
		Expect(err).NotTo(HaveOccurred())
	})
})

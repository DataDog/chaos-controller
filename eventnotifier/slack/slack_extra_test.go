// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package slack

import (
	"context"
	"os"
	"path/filepath"

	goslack "github.com/slack-go/slack"
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

var _ = Describe("slackNotifierMock Run/RunAndReturn", func() {
	It("GetUserByEmail Run callback is invoked", func() {
		m := newSlackNotifierMock(GinkgoT())
		called := false
		m.EXPECT().GetUserByEmail("a@b.com").Run(func(email string) { called = true }).Return(nil, nil)
		_, _ = m.GetUserByEmail("a@b.com")
		Expect(called).To(BeTrue())
	})

	It("GetUserByEmail RunAndReturn works", func() {
		m := newSlackNotifierMock(GinkgoT())
		m.EXPECT().GetUserByEmail("a@b.com").RunAndReturn(func(email string) (*goslack.User, error) { return nil, nil })
		_, err := m.GetUserByEmail("a@b.com")
		Expect(err).NotTo(HaveOccurred())
	})

	It("PostMessage Run callback is invoked", func() {
		m := newSlackNotifierMock(GinkgoT())
		called := false
		m.EXPECT().PostMessage("chan").Run(func(channelID string, options ...goslack.MsgOption) { called = true }).Return("", "", nil)
		_, _, _ = m.PostMessage("chan")
		Expect(called).To(BeTrue())
	})

	It("PostMessage RunAndReturn works", func() {
		m := newSlackNotifierMock(GinkgoT())
		m.EXPECT().PostMessage("chan").RunAndReturn(func(channelID string, options ...goslack.MsgOption) (string, string, error) { return "", "", nil })
		_, _, err := m.PostMessage("chan")
		Expect(err).NotTo(HaveOccurred())
	})
})

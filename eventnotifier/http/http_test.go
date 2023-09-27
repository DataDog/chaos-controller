// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package http_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/core/v1"
	coretypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/eventnotifier/http"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/mocks"
)

var _ = Describe("Http", func() {
	var (
		commonConfig types.NotifiersCommonConfig
		httpConfig   NotifierHTTPConfig
		logger       *zap.SugaredLogger
	)

	BeforeEach(func() {
		commonConfig = types.NotifiersCommonConfig{}
		httpConfig = NotifierHTTPConfig{}
		logger = zaptest.NewLogger(GinkgoT()).Sugar()
	})

	Describe("New", func() {
		It("returns error when no URL is provided", func() {
			_, err := New(commonConfig, httpConfig, logger)

			Expect(err).To(MatchError("notifier http: missing URL"))
		})

		Describe("with url", func() {
			BeforeEach(func() {
				httpConfig.URL = "some.url"
			})

			It("returns a notifier", func() {
				notifier, err := New(commonConfig, httpConfig, logger)

				Expect(err).ToNot(HaveOccurred())
				Expect(notifier).ToNot(BeNil())
				Expect(notifier.GetNotifierName()).To(Equal("http"))
			})

			It("returns an error on invalid headers", func() {
				httpConfig.Headers = []string{"key:value", "invalid"}

				notifier, err := New(commonConfig, httpConfig, logger)

				Expect(err).To(MatchError("notifier http: invalid headers in headers file: invalid headers: Must be in the format: key:value, found invalid"))
				Expect(notifier).To(BeNil())
			})
		})
	})

	Describe("Notify", func() {
		var (
			notifier           *Notifier
			wrappedHandlerFunc *fakeHTTPHandler
		)

		BeforeEach(func() {
			wrappedHandlerFunc = &fakeHTTPHandler{}
			testServer := httptest.NewServer(wrappedHandlerFunc)
			DeferCleanup(testServer.Close)

			httpConfig.URL = testServer.URL
		})

		JustBeforeEach(func() {
			var err error
			notifier, err = New(commonConfig, httpConfig, logger)

			Expect(err).ToNot(HaveOccurred())
			Expect(notifier).ToNot(BeNil())
		})

		It("returns server error", func() {
			wrappedHandlerFunc.writeHTTPStatusCode = http.StatusInternalServerError

			err := notifier.Notify(v1beta1.Disruption{}, v1.Event{}, types.NotificationInfo)

			Expect(err).To(MatchError("http notifier: receiving 500 status code from sent notification"))
		})

		Describe("body sent", func() {
			It("post default body", func() {
				err := notifier.Notify(v1beta1.Disruption{}, v1.Event{}, types.NotificationInfo)

				Expect(err).ToNot(HaveOccurred())
				wrappedHandlerFunc.ExpectBodyFromFile("post_default_body.json")
			})

			Describe("with details", func() {
				BeforeEach(func() {
					httpConfig.HasDetails = true
				})

				It("post body with details", func() {
					err := notifier.Notify(v1beta1.Disruption{}, v1.Event{}, types.NotificationInfo)

					Expect(err).ToNot(HaveOccurred())
					wrappedHandlerFunc.ExpectBodyFromFile("post_body_with_details.json")
				})

				Describe("with targets", func() {
					var dis v1beta1.Disruption

					BeforeEach(func() {
						dis = v1beta1.Disruption{
							Status: v1beta1.DisruptionStatus{
								TargetInjections: v1beta1.TargetInjections{
									"target-podA": v1beta1.TargetInjection{
										InjectorPodName: "chaos-podA",
									},
									"target-podB": v1beta1.TargetInjection{
										InjectorPodName: "chaos-podB",
									},
								},
							},
						}
					})

					It("post body with targets", func() {
						err := notifier.Notify(dis, v1.Event{}, types.NotificationInfo)

						Expect(err).ToNot(HaveOccurred())
						wrappedHandlerFunc.ExpectBodyFromFile("post_body_with_targets.json")
					})

					Describe("with k8s client", func() {
						var k8sClientMock *mocks.K8SClientMock

						BeforeEach(func() {
							k8sClientMock = mocks.NewK8SClientMock(GinkgoT())

							commonConfig.Client = k8sClientMock
						})

						It("post body with targets and pod details", func() {
							k8sClientMock.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, nn coretypes.NamespacedName, o client.Object, opt ...client.GetOption) error {
								pod := o.(*v1.Pod)
								// we just create random labels that can be easily identified by pod
								pod.Labels = map[string]string{
									"label-name":      nn.Name,
									"label-namespace": nn.Namespace,
								}
								return nil
							}).Twice()

							err := notifier.Notify(dis, v1.Event{}, types.NotificationInfo)

							Expect(err).ToNot(HaveOccurred())
							wrappedHandlerFunc.ExpectBodyFromFile("post_body_with_targets_and_pods_details.json")
						})
					})
				})
			})
		})
	})
})

// fakeHTTPHandler store request body for later comparison
// it also writes provided response http status code
type fakeHTTPHandler struct {
	writeHTTPStatusCode int
	body                []byte
}

// ServeHTTP match http.Handler interface
func (d *fakeHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer GinkgoRecover()
	GinkgoHelper()

	body, err := io.ReadAll(r.Body)
	Expect(err).ToNot(HaveOccurred())

	d.body = body

	if d.writeHTTPStatusCode != 0 {
		w.WriteHeader(d.writeHTTPStatusCode)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

// ExpectBodyFromFile read provided filename in testdata to perform comparison with stored body
func (d *fakeHTTPHandler) ExpectBodyFromFile(filename string) {
	defer GinkgoRecover()
	GinkgoHelper()

	bodyBytes, err := os.ReadFile(filepath.Join("testdata", filename))

	Expect(err).ToNot(HaveOccurred())

	Expect(d.body).To(MatchJSON(bodyBytes))
}

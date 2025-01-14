// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/eventnotifier/utils"
	"github.com/DataDog/jsonapi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notifier", func() {
	var (
		logger                    *zap.SugaredLogger
		defaultDisruptionName     = "disruption-name"
		defaultDisruptionCronName = "disruption-cron-name"
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT()).Sugar()
	})

	Describe("New", func() {
		Describe("success cases", func() {
			DescribeTable("should return a notifier instance", func(config Config) {
				// Act
				notifier, err := New(types.NotifiersCommonConfig{}, config, logger)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
				Expect(notifier).ShouldNot(BeNil())
				Expect(notifier.disruptionConfig).Should(Equal(config.Disruption))
				Expect(notifier.disruptionCronConfig).Should(Equal(config.DisruptionCron))
			},
				Entry("with disruption enabled", Config{
					Disruption: DisruptionConfig{
						Enabled: true,
						URL:     "http://localhost/disruption",
					},
				}),
				Entry("with disruption cron enabled", Config{
					DisruptionCron: DisruptionCronConfig{
						Enabled: true,
						URL:     "http://localhost/disruption-cron",
					},
				}),
				Entry("with both disruption and disruption cron enabled", Config{
					Disruption: DisruptionConfig{
						Enabled: true,
						URL:     "http://localhost/disruption",
					},
					DisruptionCron: DisruptionCronConfig{
						Enabled: true,
						URL:     "http://localhost/disruption-cron",
					},
				}),
			)
			Context("with disruption enabled", func() {
				It("should return a notifier instance", func() {
					// Arrange
					config := Config{
						Disruption: DisruptionConfig{
							Enabled: true,
							URL:     "http://localhost/disruption",
						},
					}

					// Act
					notifier, err := New(types.NotifiersCommonConfig{}, config, logger)

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
					Expect(notifier).ShouldNot(BeNil())
					Expect(notifier.disruptionConfig).Should(Equal(DisruptionConfig{
						Enabled: true,
						URL:     "http://localhost/disruption",
					}))
				})
			})

			Context("with disruptionCron enabled", func() {
				It("should return a notifier instance", func() {
					// Arrange
					config := Config{
						DisruptionCron: DisruptionCronConfig{
							Enabled: true,
							URL:     "http://localhost/disruption-cron",
						},
					}

					// Act
					notifier, err := New(types.NotifiersCommonConfig{}, config, logger)

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
					Expect(notifier).ShouldNot(BeNil())
					Expect(notifier.disruptionConfig).Should(Equal(DisruptionConfig{
						Enabled: false,
						URL:     "",
					}))
					Expect(notifier.disruptionCronConfig).Should(Equal(DisruptionCronConfig{
						Enabled: true,
						URL:     "http://localhost/disruption-cron",
					}))
				})
			})
		})

		Describe("error cases", func() {
			DescribeTable("the URL is not defined", func(config Config, expectedError string) {
				// Act
				notifier, err := New(types.NotifiersCommonConfig{}, config, logger)

				// Assert
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError(expectedError))
				Expect(notifier).Should(BeNil())
			},
				Entry("with disruption enabled", Config{
					Disruption: DisruptionConfig{
						Enabled: true,
					},
				}, "http notifier: missing URL for disruption notifications"),
				Entry("with disruption cron enabled", Config{
					DisruptionCron: DisruptionCronConfig{
						Enabled: true,
					},
				}, "http notifier: missing URL URL for disruption cron notifications"),
			)
		})
	})

	Describe("Config", func() {
		DescribeTable("IsEnabled", func(config Config, expectedResult bool) {
			// Act
			result := config.IsEnabled()

			// Assert
			Expect(result).Should(Equal(expectedResult))

		},
			Entry("with disruption-cron enable it should return true", Config{
				DisruptionCron: DisruptionCronConfig{
					Enabled: true,
					URL:     "",
				},
			}, true),
			Entry("with disruption enable it should return true", Config{
				Disruption: DisruptionConfig{
					Enabled: true,
					URL:     "",
				},
			}, true),
			Entry("with both enabled it should return true", Config{
				Disruption: DisruptionConfig{
					Enabled: true,
					URL:     "",
				},
				DisruptionCron: DisruptionCronConfig{
					Enabled: true,
					URL:     "",
				},
			}, true),
			Entry("with none enabled it should return false", Config{}, false),
		)
	})

	Describe("Notify", func() {
		Describe("success cases", func() {
			DescribeTable("with an unsupported object", func(obj client.Object) {
				// Arrange
				notifier := Notifier{
					common: types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					client: &http.Client{},
					logger: logger,
				}

				// Act
				err := notifier.Notify(obj, corev1.Event{}, types.NotificationInfo)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("with a pod object", &corev1.Pod{}),
				Entry("with a node object", &corev1.Node{}),
				Entry("with a service object", &corev1.Service{}),
				Entry("with a namespace object", &corev1.Namespace{}),
			)

			DescribeTable("with supported object", func(objKind string) {
				// Arrange
				var obj client.Object

				expectedUID := uuid.New().String()
				expectedNotificationType := types.NotificationInfo

				event := corev1.Event{
					InvolvedObject: corev1.ObjectReference{
						Kind: objKind,
					},
					Reason: string(v1beta1.EventDisruptionFinished),
				}

				switch objKind {
				case v1beta1.DisruptionKind:
					obj = &v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							UID:  apimachineryTypes.UID(expectedUID),
							Name: defaultDisruptionName,
						},
					}
				case v1beta1.DisruptionCronKind:
					obj = &v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							UID:  apimachineryTypes.UID(expectedUID),
							Name: defaultDisruptionCronName,
						},
					}
				}

				expectedNotifierEvent := NotifierEvent{
					ID:                 expectedUID,
					NotificationTitle:  utils.BuildHeaderMessageFromObjectEvent(obj, event, expectedNotificationType),
					NotificationType:   expectedNotificationType,
					EventMessage:       utils.BuildBodyMessageFromObjectEvent(obj, event, false),
					InvolvedObjectKind: obj.GetObjectKind().GroupVersionKind().Kind,
					Cluster:            "cluster-name",
					Namespace:          obj.GetNamespace(),
					UserGroups:         "null",
					EventReason:        v1beta1.EventDisruptionFinished,
				}

				svrDisruption := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					By("sending a POST request")
					Expect(r.Method).To(Equal(http.MethodPost))

					expectedDisruptionStr, err := json.Marshal(obj)
					Expect(err).ToNot(HaveOccurred())

					expectedDisruptionEvent := DisruptionEvent{
						TargetsCount:  0,
						Manifest:      string(expectedDisruptionStr),
						NotifierEvent: expectedNotifierEvent,
					}

					By("set deprecated fields")
					expectedDisruptionEvent.Name = defaultDisruptionName
					expectedDisruptionEvent.Disruption = string(expectedDisruptionStr)
					expectedDisruptionEvent.DisruptionName = defaultDisruptionName
					expectedDisruptionEvent.NotifierEvent.TargetsCount = 0

					var disruptionEvent DisruptionEvent

					body, err := io.ReadAll(r.Body)
					Expect(err).ShouldNot(HaveOccurred())

					Expect(jsonapi.Unmarshal(body, &disruptionEvent)).To(Succeed())
					disruptionEvent.Timestamp = expectedNotifierEvent.Timestamp

					By("sending a valid disruptionevent event")
					Expect(disruptionEvent).To(Equal(expectedDisruptionEvent))
				}))

				svrDisruptionCron := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					By("sending a POST request")
					Expect(r.Method).To(Equal(http.MethodPost))

					expectedDisruptionCronStr, err := json.Marshal(obj)
					Expect(err).ToNot(HaveOccurred())

					expectedDisruptionCronEvent := DisruptionCronEvent{
						Manifest:      string(expectedDisruptionCronStr),
						NotifierEvent: expectedNotifierEvent,
					}
					expectedDisruptionCronEvent.Name = defaultDisruptionCronName

					var disruptionCronEvent DisruptionCronEvent

					body, err := io.ReadAll(r.Body)
					Expect(err).ShouldNot(HaveOccurred())

					Expect(jsonapi.Unmarshal(body, &disruptionCronEvent)).To(Succeed())
					disruptionCronEvent.Timestamp = expectedDisruptionCronEvent.Timestamp

					By("sending the correct event")
					Expect(disruptionCronEvent).To(Equal(expectedDisruptionCronEvent))
				}))

				notifier := Notifier{
					common: types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					client: &http.Client{},
					logger: logger,
					disruptionConfig: DisruptionConfig{
						Enabled: true,
						URL:     svrDisruption.URL,
					},
					disruptionCronConfig: DisruptionCronConfig{
						Enabled: true,
						URL:     svrDisruptionCron.URL,
					},
				}

				// Act
				err := notifier.Notify(obj, event, expectedNotificationType)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("with a disruption kind", v1beta1.DisruptionKind),
				Entry("with a disruptionCron kind", v1beta1.DisruptionCronKind),
			)

			When("the authentication provider is defined", func() {
				It("should set the Authorization header", func() {
					// Arrange
					svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						By("sending a POST request")
						Expect(r.Method).To(Equal(http.MethodPost))

						By("asserting the Authorization header is set")
						Expect(r.Header.Get("Authorization")).To(Equal("Bearer token"))
					}))

					mockBearerAuthTokenProvider := NewBearerAuthTokenProviderMock(GinkgoT())
					mockBearerAuthTokenProvider.EXPECT().AuthToken(mock.Anything).Return("token", nil)

					notifier := Notifier{
						client:            &http.Client{},
						logger:            logger,
						authTokenProvider: mockBearerAuthTokenProvider,
						disruptionConfig: DisruptionConfig{
							Enabled: true,
							URL:     svr.URL,
						},
					}

					// Act
					err := notifier.Notify(
						&v1beta1.Disruption{
							ObjectMeta: metav1.ObjectMeta{
								UID: apimachineryTypes.UID(uuid.New().String()),
							},
						},
						corev1.Event{
							InvolvedObject: corev1.ObjectReference{
								Kind: v1beta1.DisruptionKind,
							},
						},
						types.NotificationInfo,
					)

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			Context("with custom headers", func() {
				It("should set the custom headers", func() {
					// Arrange
					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						By("sending a POST request")
						Expect(r.Method).To(Equal(http.MethodPost))

						By("asserting the custom headers are set")
						Expect(r.Header.Get("Custom-Header")).To(Equal("custom-value"))
						Expect(r.Header.Get("Custom-Header2")).To(Equal("custom-value2"))
					}))

					notifier := Notifier{
						client: &http.Client{},
						headers: map[string]string{
							"Custom-Header":  "custom-value",
							"Custom-Header2": "custom-value2",
						},
						logger: logger,
						disruptionConfig: DisruptionConfig{
							Enabled: true,
							URL:     srv.URL,
						},
					}

					// Act
					err := notifier.Notify(
						&v1beta1.Disruption{
							ObjectMeta: metav1.ObjectMeta{
								UID: apimachineryTypes.UID(uuid.New().String()),
							},
						},
						corev1.Event{
							InvolvedObject: corev1.ObjectReference{
								Kind: v1beta1.DisruptionKind,
							},
						},
						types.NotificationInfo,
					)

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			DescribeTable("when notifier is disabled", func(notifier Notifier, object client.Object, event corev1.Event) {
				// Arrange
				httpCallCount := 0
				httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					httpCallCount++
				}))

				notifier.client = &http.Client{}
				notifier.disruptionConfig.URL = httpServer.URL
				notifier.disruptionCronConfig.URL = httpServer.URL
				notifier.logger = logger

				// Act
				err := notifier.Notify(
					object,
					event,
					types.NotificationInfo,
				)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())

				By("not send an http request")
				Expect(httpCallCount).To(Equal(0))
			},
				Entry("disruption event is disabled",
					Notifier{
						disruptionConfig: DisruptionConfig{
							Enabled: false,
						},
						disruptionCronConfig: DisruptionCronConfig{
							Enabled: true,
						},
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							UID: apimachineryTypes.UID(uuid.New().String()),
						},
					},
					corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
					},
				),
				Entry("disruption cron event is disabled",
					Notifier{
						disruptionConfig: DisruptionConfig{
							Enabled: true,
						},
						disruptionCronConfig: DisruptionCronConfig{
							Enabled: false,
						},
					},
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							UID: apimachineryTypes.UID(uuid.New().String()),
						},
					},
					corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionCronKind,
						},
					},
				),
			)
		})

		Describe("error cases", func() {
			When("the request fails", func() {
				DescribeTable("it should return the error", func(httpStatus int, objKind string) {
					// Arrange
					var obj client.Object

					expectedUID := uuid.New().String()
					expectedNoticationType := types.NotificationInfo

					event := corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind: objKind,
						},
					}

					switch objKind {
					case v1beta1.DisruptionKind:
						obj = &v1beta1.Disruption{
							ObjectMeta: metav1.ObjectMeta{
								UID:  apimachineryTypes.UID(expectedUID),
								Name: defaultDisruptionName,
							},
						}
					case v1beta1.DisruptionCronKind:
						obj = &v1beta1.DisruptionCron{
							ObjectMeta: metav1.ObjectMeta{
								UID:  apimachineryTypes.UID(expectedUID),
								Name: defaultDisruptionCronName,
							},
						}
					}

					svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						By("sending a POST request")
						Expect(r.Method).To(Equal(http.MethodPost))

						w.WriteHeader(httpStatus)
					}))

					notifier := Notifier{
						common: types.NotifiersCommonConfig{
							ClusterName: "cluster-name",
						},
						client: &http.Client{},
						logger: logger,
						disruptionConfig: DisruptionConfig{
							Enabled: true,
							URL:     svr.URL,
						},
						disruptionCronConfig: DisruptionCronConfig{
							Enabled: true,
							URL:     svr.URL,
						},
					}

					// Act
					err := notifier.Notify(obj, event, expectedNoticationType)

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(fmt.Sprintf("http notifier: receiving %d status code from sent notification", httpStatus)))
				},
					Entry("500 HTTP error with a disruption kind event", http.StatusInternalServerError, v1beta1.DisruptionKind),
					Entry("300 HTTP error with a disruption kind event", http.StatusMultipleChoices, v1beta1.DisruptionKind),
					Entry("302 HTTP error with a disruption kind event", http.StatusFound, v1beta1.DisruptionKind),
					Entry("101 HTTP error with a disruption kind event", http.StatusSwitchingProtocols, v1beta1.DisruptionKind),
					Entry("500 HTTP error with a disruptionCron kind event", http.StatusInternalServerError, v1beta1.DisruptionCronKind),
				)
			})

			When("the authentication provider fails", func() {
				It("should return the error", func() {
					// Arrange
					mockBearerAuthTokenProvider := NewBearerAuthTokenProviderMock(GinkgoT())
					mockBearerAuthTokenProvider.EXPECT().AuthToken(mock.Anything).Return("", fmt.Errorf("error"))

					notifier := Notifier{
						client:            &http.Client{},
						logger:            logger,
						authTokenProvider: mockBearerAuthTokenProvider,
						disruptionConfig: DisruptionConfig{
							Enabled: true,
						},
					}

					// Act
					err := notifier.Notify(&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							UID: apimachineryTypes.UID(uuid.New().String()),
						},
					}, corev1.Event{}, types.NotificationInfo)

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("http notifier: unable to retrieve auth token through helper: error"))
				})
			})
		})
	})
})

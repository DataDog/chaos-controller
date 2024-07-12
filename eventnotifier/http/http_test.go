// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

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
	zaplog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/jsonapi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryTypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Notifier", func() {
	var (
		logger *zap.SugaredLogger
	)

	BeforeEach(func() {
		var err error
		logger, err = zaplog.NewZapLogger()
		Expect(err).ShouldNot(HaveOccurred())
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
					url:    "http://localhost",
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
							Name: "disruption-name",
						},
					}
				case v1beta1.DisruptionCronKind:
					obj = &v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							UID:  apimachineryTypes.UID(expectedUID),
							Name: "disruption-cron-name",
						},
					}
				}

				expectedNotifierEvent := NotifierEvent{
					ID:                 expectedUID,
					NotificationTitle:  utils.BuildHeaderMessageFromObjectEvent(obj, event, expectedNoticationType),
					NotificationType:   expectedNoticationType,
					EventMessage:       utils.BuildBodyMessageFromObjectEvent(obj, event, false),
					InvolvedObjectKind: obj.GetObjectKind().GroupVersionKind().Kind,
					Cluster:            "cluster-name",
					Namespace:          obj.GetNamespace(),
					UserGroups:         "null",
				}

				svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					By("sending a POST request")
					Expect(r.Method).To(Equal(http.MethodPost))

					switch objKind {
					case v1beta1.DisruptionKind:
						expectedDisruptionStr, err := json.Marshal(obj)
						Expect(err).ToNot(HaveOccurred())

						expectedNotifierEvent.DisruptionEvent = &DisruptionEvent{
							TargetsCount:   0,
							DisruptionName: "disruption-name",
							Disruption:     string(expectedDisruptionStr),
						}

						var notifierEvent NotifierEvent

						body, err := io.ReadAll(r.Body)
						Expect(err).ShouldNot(HaveOccurred())

						Expect(jsonapi.Unmarshal(body, &notifierEvent)).To(Succeed())
						notifierEvent.Timestamp = expectedNotifierEvent.Timestamp

						By("sending the correct event")
						Expect(notifierEvent).To(Equal(expectedNotifierEvent))
					case v1beta1.DisruptionCronKind:
						expectedDisruptionCronStr, err := json.Marshal(obj)
						Expect(err).ToNot(HaveOccurred())

						expectedNotifierEvent.DisruptionCronEvent = &DisruptionCronEvent{
							DisruptionCronName: "disruption-cron-name",
							DisruptionCron:     string(expectedDisruptionCronStr),
						}

						var notifierEvent NotifierEvent

						body, err := io.ReadAll(r.Body)
						Expect(err).ShouldNot(HaveOccurred())

						Expect(jsonapi.Unmarshal(body, &notifierEvent)).To(Succeed())
						notifierEvent.Timestamp = expectedNotifierEvent.Timestamp

						By("sending the correct event")
						Expect(notifierEvent).To(Equal(expectedNotifierEvent))
					}
				}))

				notifier := Notifier{
					common: types.NotifiersCommonConfig{
						ClusterName: "cluster-name",
					},
					client: &http.Client{},
					url:    svr.URL,
					logger: logger,
				}

				// Act
				err := notifier.Notify(obj, event, expectedNoticationType)

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
						url:               svr.URL,
						logger:            logger,
						authTokenProvider: mockBearerAuthTokenProvider,
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
					svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						By("sending a POST request")
						Expect(r.Method).To(Equal(http.MethodPost))

						By("asserting the custom headers are set")
						Expect(r.Header.Get("Custom-Header")).To(Equal("custom-value"))
						Expect(r.Header.Get("Custom-Header2")).To(Equal("custom-value2"))
					}))

					notifier := Notifier{
						client: &http.Client{},
						url:    svr.URL,
						logger: logger,
						headers: map[string]string{
							"Custom-Header":  "custom-value",
							"Custom-Header2": "custom-value2",
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
								Name: "disruption-name",
							},
						}
					case v1beta1.DisruptionCronKind:
						obj = &v1beta1.DisruptionCron{
							ObjectMeta: metav1.ObjectMeta{
								UID:  apimachineryTypes.UID(expectedUID),
								Name: "disruption-cron-name",
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
						url:    svr.URL,
						logger: logger,
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
						url:               "http://localhost",
						logger:            logger,
						authTokenProvider: mockBearerAuthTokenProvider,
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

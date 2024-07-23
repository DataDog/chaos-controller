// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package eventbroadcaster

import (
	"context"
	"fmt"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier"
	notifTypes "github.com/DataDog/chaos-controller/eventnotifier/types"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NotifierSink", func() {

	var (
		mockClient      *mocks.K8SClientMock
		mockManager     *mocks.ManagerMock
		mockBroadcaster *mocks.EventBroadcasterMock
		logger          *zap.SugaredLogger
	)

	BeforeEach(func() {
		mockClient = mocks.NewK8SClientMock(GinkgoT())
		mockManager = mocks.NewManagerMock(GinkgoT())
		mockBroadcaster = mocks.NewEventBroadcasterMock(GinkgoT())

		logger = zaptest.NewLogger(GinkgoT()).Sugar()
	})

	Describe("RegisterNotifierSinks", func() {
		Describe("success cases", func() {
			It("should succeed", func() {
				// Arrange
				By("getting the client and config")
				mockManager.EXPECT().GetClient().Return(mockClient)
				mockManager.EXPECT().GetConfig().Return(&rest.Config{})

				mockBroadcaster.EXPECT().StartRecordingToSink(mock.Anything).Return(nil)

				firstNotifierMock := eventnotifier.NewNotifierMock(GinkgoT())
				firstNotifierMock.EXPECT().GetNotifierName().Return("test")
				secondNotifierMock := eventnotifier.NewNotifierMock(GinkgoT())
				secondNotifierMock.EXPECT().GetNotifierName().Return("test")
				notifiers := []eventnotifier.Notifier{
					firstNotifierMock,
					secondNotifierMock,
				}

				// Act
				RegisterNotifierSinks(mockManager, mockBroadcaster, notifiers, logger)
			})
		})
	})

	Describe("Create", func() {
		Describe("success cases", func() {
			DescribeTable("when the Notifier is not triggered", func(objectKind string) {
				// Arrange
				event := corev1.Event{
					InvolvedObject: corev1.ObjectReference{
						Kind: objectKind,
					},
					Type: corev1.EventTypeNormal,
				}

				if objectKind == v1beta1.DisruptionKind || objectKind == v1beta1.DisruptionCronKind {
					mockClient.EXPECT().Get(
						mock.Anything, types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, mock.Anything,
					).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
						switch objectKind {
						case v1beta1.DisruptionKind:
							*object.(*v1beta1.Disruption) = v1beta1.Disruption{}
						case v1beta1.DisruptionCronKind:
							*object.(*v1beta1.DisruptionCron) = v1beta1.DisruptionCron{}
						}

						return nil
					})

				}

				notifierMock := eventnotifier.NewNotifierMock(GinkgoT())

				notifierSink := NotifierSink{
					client:   mockClient,
					notifier: notifierMock,
					logger:   logger,
				}

				// Action
				_, err := notifierSink.Create(&event)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("a Manifest kind",
					v1beta1.DisruptionKind,
				),
				Entry("a Manifest kind",
					v1beta1.DisruptionCronKind,
				),
				Entry("a unknown kind",
					"Unknown",
				),
			)

			DescribeTable("when the Notifier is triggered", func(event corev1.Event, expectedObject client.Object, expectedNotifType notifTypes.NotificationType) {
				// Arrange
				By("expecting the client to get the object")
				mockClient.EXPECT().Get(
					mock.Anything, types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, mock.Anything,
				).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
					switch expectedObject.(type) {
					case *v1beta1.Disruption:
						disruption := expectedObject.(*v1beta1.Disruption)
						*object.(*v1beta1.Disruption) = *disruption
					case *v1beta1.DisruptionCron:
						disruptionCron := expectedObject.(*v1beta1.DisruptionCron)
						*object.(*v1beta1.DisruptionCron) = *disruptionCron
					}

					return nil
				})

				notifierMock := eventnotifier.NewNotifierMock(GinkgoT())

				By("expecting the notifier to be called")
				notifierMock.EXPECT().Notify(expectedObject, event, expectedNotifType).Return(nil)

				notifierSink := NotifierSink{
					client:   mockClient,
					notifier: notifierMock,
					logger:   logger,
				}

				// Action
				_, err := notifierSink.Create(&event)

				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			},
				Entry("a Manifest kind event with a warning type",
					corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
						Type: corev1.EventTypeWarning,
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationWarning,
				),
				Entry("a Manifest kind event with a normal notifiable type",
					corev1.Event{
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationInfo,
				),
				Entry("a Manifest kind event with a normal type with a target node recovered reason",
					corev1.Event{
						Reason: string(v1beta1.EventTargetNodeRecoveredState),
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationSuccess,
				),
				Entry("a Manifest kind event with a normal type and a Finished reason",
					corev1.Event{
						Reason: string(v1beta1.EventDisruptionFinished),
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationCompletion,
				),
				Entry("a Manifest kind event with a normal type and a DurationOver reason",
					corev1.Event{
						Reason: string(v1beta1.EventDisruptionDurationOver),
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationCompletion,
				),
				Entry("a Manifest kind event with a normal type and a GCOver reason",
					corev1.Event{
						Reason: string(v1beta1.EventDisruptionGCOver),
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.Disruption{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationCompletion,
				),
				Entry("a Manifest kind event with a warning type",
					corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind:      v1beta1.DisruptionCronKind,
							Namespace: "fake-namespace",
							Name:      "test",
						},
						Type: corev1.EventTypeWarning,
					},
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationWarning,
				),
				Entry("a Manifest kind event with a normal notifiable type",
					corev1.Event{
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionCronComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionCronKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationInfo,
				),
				Entry("a Manifest kind event with a normal type and a Deleted reason",
					corev1.Event{
						Reason: string(v1beta1.EventDisruptionCronDeleted),
						Source: corev1.EventSource{
							Component: v1beta1.SourceDisruptionCronComponent,
						},
						InvolvedObject: corev1.ObjectReference{
							Kind: v1beta1.DisruptionCronKind,
						},
						Type: corev1.EventTypeNormal,
					},
					&v1beta1.DisruptionCron{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "fake-namespace",
						},
					},
					notifTypes.NotificationInfo,
				),
			)

			When("the client fails to get the object", func() {
				DescribeTable("it should not return an error", func(objectKind string) {
					// Arrange
					expectedUID := uuid.New().String()
					expectedNamespace := fmt.Sprintf("namespace-%s", objectKind)
					expectedName := fmt.Sprintf("name-%s", objectKind)
					expectedNotificationType := notifTypes.NotificationInfo

					event := corev1.Event{
						ObjectMeta: metav1.ObjectMeta{},
						InvolvedObject: corev1.ObjectReference{
							Kind:      objectKind,
							Namespace: expectedNamespace,
							Name:      expectedName,
							UID:       types.UID(expectedUID),
						},
						Type: corev1.EventTypeNormal,
					}

					var expectedObject client.Object

					switch objectKind {
					case v1beta1.DisruptionKind:
						expectedObject = &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      expectedName,
								Namespace: expectedNamespace,
								UID:       types.UID(expectedUID),
							},
						}
						event.Source.Component = v1beta1.SourceDisruptionComponent
					case v1beta1.DisruptionCronKind:
						expectedObject = &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:      expectedName,
								Namespace: expectedNamespace,
								UID:       types.UID(expectedUID),
							},
						}
						event.Source.Component = v1beta1.SourceDisruptionCronComponent
					}

					By("expecting the client to get the object")
					mockClient.EXPECT().Get(
						mock.Anything, types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, mock.Anything,
					).Return(fmt.Errorf("error"))

					By("expecting the notifier to be called")
					notifierMock := eventnotifier.NewNotifierMock(GinkgoT())
					notifierMock.EXPECT().Notify(expectedObject, event, expectedNotificationType).Return(nil)

					notifierSink := NotifierSink{
						client:   mockClient,
						notifier: notifierMock,
						logger:   logger,
					}

					// Act
					_, err := notifierSink.Create(&event)

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				},
					Entry("with a Manifest kind event",
						v1beta1.DisruptionKind,
					),
					Entry("with a Manifest kind event",
						v1beta1.DisruptionCronKind,
					),
				)
			})
		})

		Describe("error cases", func() {
			When("the event type is not supported", func() {
				DescribeTable("it should not return an error", func(objectKind string) {
					// Arrange
					event := corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind: objectKind,
						},
						Type: "unsupported",
					}

					notifierMock := eventnotifier.NewNotifierMock(GinkgoT())

					mockClient.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil)

					notifierSink := NotifierSink{
						client:   mockClient,
						notifier: notifierMock,
						logger:   logger,
					}

					// Act
					_, err := notifierSink.Create(&event)

					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				},
					Entry("with a Manifest kind event",
						v1beta1.DisruptionKind,
					),
					Entry("with a Manifest kind event",
						v1beta1.DisruptionCronKind,
					),
				)
			})

			When("the notifier fails to notify", func() {
				DescribeTable("it should return an error", func(objectKind string) {
					// Arrange
					event := corev1.Event{
						InvolvedObject: corev1.ObjectReference{
							Kind: objectKind,
						},
						Type: corev1.EventTypeWarning,
					}

					By("expecting the client to get the object")
					mockClient.EXPECT().Get(
						mock.Anything, types.NamespacedName{Namespace: event.InvolvedObject.Namespace, Name: event.InvolvedObject.Name}, mock.Anything,
					).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error { //nolint:ineffassign,staticcheck
						switch objectKind {
						case v1beta1.DisruptionKind:
							//nolint:ineffassign,staticcheck
							object = &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}
						case v1beta1.DisruptionCronKind:
							//nolint:ineffassign,staticcheck
							object = &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}
						}

						return nil
					})

					notifierMock := eventnotifier.NewNotifierMock(GinkgoT())

					By("expecting the notifier to be called")
					notifierMock.EXPECT().Notify(mock.Anything, event, mock.Anything).Return(fmt.Errorf("notify error"))

					notifierSink := NotifierSink{
						client:   mockClient,
						notifier: notifierMock,
						logger:   logger,
					}

					// Action
					_, err := notifierSink.Create(&event)

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err).Should(MatchError("notifier: failed to notify: notify error"))

				},
					Entry("with a Disruption kind event",
						v1beta1.DisruptionKind,
					),
					Entry("with a Disruption Cron kind event",
						v1beta1.DisruptionCronKind,
					),
				)
			})
		})
	})
})

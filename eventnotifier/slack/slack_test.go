// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package slack

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/eventnotifier/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Slack Notifier", func() {
	Describe("Notify", func() {
		var (
			defaultClusterName         = "cluster-name"
			defaultUserInfoUserName    = "email@test.com"
			defaultMirrorSlackChanelID = "chaos-notif"
			slackNotifMock             *slackNotifierMock
			notifier                   Notifier
		)

		BeforeEach(func() {
			slackNotifMock = newSlackNotifierMock(GinkgoT())
			notifier = Notifier{
				client: slackNotifMock,
				common: types.NotifiersCommonConfig{
					ClusterName: defaultClusterName,
				},
				config: NotifierSlackConfig{
					Enabled:       true,
					TokenFilepath: "/path/to/token",
				},
			}
		})

		Describe("success cases", func() {
			Context("with a mirror slack channel", func() {
				BeforeEach(func() {
					notifier.config.MirrorSlackChannelID = defaultMirrorSlackChanelID
				})

				Context("with an info notification", func() {
					Context("with a disruption", func() {
						It("should send a slack message to the mirror slack channel", func(ctx SpecContext) {
							// Arrange
							slackNotifMock.EXPECT().PostMessage(
								defaultMirrorSlackChanelID,
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
							).Return("", "", nil).Once()

							slackNotifMock.AssertNotCalled(GinkgoT(), "GetUserByEmail")

							// Act
							err := notifier.Notify(ctx, &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, corev1.Event{}, types.NotificationInfo)

							// Assert
							Expect(err).ShouldNot(HaveOccurred())
						})
					})
				})

				Context("with all other notifications", func() {
					DescribeTable("should send a slack message to the mirror slack channel and the user chanel", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
						// Arrange
						expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
						switch d := obj.(type) {
						case *v1beta1.Disruption:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
						case *v1beta1.DisruptionCron:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
						}

						By("sending the message to the mirror slack channel")
						slackNotifMock.EXPECT().PostMessage(
							defaultMirrorSlackChanelID,
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
						).Return("", "", nil).Once()

						expectedSlackUser := &slack.User{
							ID:   uuid.New().String(),
							Name: expectedUserInfo.Username,
						}
						slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

						By("sending the message to the slack user")
						slackNotifMock.EXPECT().PostMessage(
							expectedSlackUser.ID,
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
						).Return("", "", nil).Once()

						// Act
						err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					},
						Entry("a success notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationSuccess),
						Entry("a completion notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationCompletion),
						Entry("a unknown notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationUnknown),
						Entry("a warning notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationWarning),
						Entry("an error notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationError),
						Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationInfo),
						Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationCompletion),
						Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationUnknown),
						Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationWarning),
						Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationError),
					)

					Context("with a slack configuration defined in the resource", func() {
						DescribeTable("should send a slack message to the mirror, the user chanel and the custom channel", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
							// Arrange
							expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
							switch d := obj.(type) {
							case *v1beta1.Disruption:
								Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							case *v1beta1.DisruptionCron:
								Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							}

							expectedReportingSlackChannel := "custom-slack-channel"
							expectedReporting := &v1beta1.Reporting{
								SlackChannel: expectedReportingSlackChannel,
								Purpose:      "lorem",
							}

							switch d := obj.(type) {
							case *v1beta1.Disruption:
								d.Spec.Reporting = expectedReporting
								obj = d
							case *v1beta1.DisruptionCron:
								d.Spec.Reporting = expectedReporting
								obj = d
							}

							By("sending the message to the mirror slack channel")
							slackNotifMock.EXPECT().PostMessage(
								defaultMirrorSlackChanelID,
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
							).Return("", "", nil).Once()

							By("sending the message to the slack chanel from the reporting configuration")
							slackNotifMock.EXPECT().PostMessage(
								expectedReportingSlackChannel,
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
							).Return("", "", nil).Once()

							expectedSlackUser := &slack.User{
								ID:   uuid.New().String(),
								Name: expectedUserInfo.Username,
							}
							slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

							By("sending the message to the slack user")
							slackNotifMock.EXPECT().PostMessage(
								expectedSlackUser.ID,
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
							).Return("", "", nil).Once()

							// Act
							err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

							// Assert
							Expect(err).ShouldNot(HaveOccurred())
						},
							Entry("a success notification with a disruption", &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, types.NotificationSuccess),
							Entry("a completion notification with a disruption", &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, types.NotificationCompletion),
							Entry("a unknown notification with a disruption", &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, types.NotificationUnknown),
							Entry("a warning notification with a disruption", &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, types.NotificationWarning),
							Entry("an error notification with a disruption", &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, types.NotificationError),
							Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationInfo),
							Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationCompletion),
							Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationUnknown),
							Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationWarning),
							Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationError),
						)

						When("the message to the custom slack channel fails", func() {
							DescribeTable("should send a slack message to the mirror, the user chanel and the custom channel and not failed", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
								// Arrange
								expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
								switch d := obj.(type) {
								case *v1beta1.Disruption:
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								case *v1beta1.DisruptionCron:
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								}

								expectedReportingSlackChannel := "custom-slack-channel"
								expectedReporting := &v1beta1.Reporting{
									SlackChannel: expectedReportingSlackChannel,
								}

								switch d := obj.(type) {
								case *v1beta1.Disruption:
									d.Spec.Reporting = expectedReporting
									obj = d
								case *v1beta1.DisruptionCron:
									d.Spec.Reporting = expectedReporting
									obj = d
								}

								By("sending the message to the mirror slack channel")
								slackNotifMock.EXPECT().PostMessage(
									defaultMirrorSlackChanelID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								By("sending the message to the slack chanel from the reporting configuration")
								slackNotifMock.EXPECT().PostMessage(
									expectedReportingSlackChannel,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", fmt.Errorf("could not send message to reporting slack channel")).Once()

								expectedSlackUser := &slack.User{
									ID:   uuid.New().String(),
									Name: expectedUserInfo.Username,
								}
								slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

								By("sending the message to the slack user")
								slackNotifMock.EXPECT().PostMessage(
									expectedSlackUser.ID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								// Act
								err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

								// Assert
								Expect(err).ShouldNot(HaveOccurred())
							},
								Entry("a success notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationSuccess),
								Entry("a completion notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationError),
								Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationInfo),
								Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationError),
							)
						})

						When("the message to the mirror slack channel fails", func() {
							DescribeTable("should send a slack message the user chanel and the custom channel, but not returns an error", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
								// Arrange
								expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}

								switch d := obj.(type) {
								case *v1beta1.Disruption:
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								case *v1beta1.DisruptionCron:
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								}

								expectedReportingSlackChannel := "custom-slack-channel"
								expectedReporting := &v1beta1.Reporting{
									SlackChannel: expectedReportingSlackChannel,
								}

								switch d := obj.(type) {
								case *v1beta1.Disruption:
									d.Spec.Reporting = expectedReporting
									obj = d
								case *v1beta1.DisruptionCron:
									d.Spec.Reporting = expectedReporting
									obj = d
								}

								By("sending the message to the mirror slack channel")
								slackNotifMock.EXPECT().PostMessage(
									defaultMirrorSlackChanelID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", fmt.Errorf("could not send error the the mirror channel")).Once()

								By("sending the message to the slack chanel from the reporting configuration")
								slackNotifMock.EXPECT().PostMessage(
									expectedReportingSlackChannel,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								expectedSlackUser := &slack.User{
									ID:   uuid.New().String(),
									Name: expectedUserInfo.Username,
								}
								slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

								By("sending the message to the slack user")
								slackNotifMock.EXPECT().PostMessage(
									expectedSlackUser.ID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								// Act
								err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

								// Assert
								Expect(err).ShouldNot(HaveOccurred())
							},
								Entry("a success notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationSuccess),
								Entry("a completion notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationError),
								Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationInfo),
								Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationError),
							)

						})
					})
				})
			})

			Context("without a mirror slack channel", func() {
				Context("with an info notification", func() {
					Context("with a disruption", func() {
						It("should noy send any message", func(ctx SpecContext) {
							// Arrange
							slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage")
							slackNotifMock.AssertNotCalled(GinkgoT(), "GetUserByEmail")

							// Act
							err := notifier.Notify(ctx, &v1beta1.Disruption{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionKind,
								},
							}, corev1.Event{}, types.NotificationInfo)

							// Assert
							Expect(err).ShouldNot(HaveOccurred())
						})
					})
				})

				Context("with all other notifications", func() {
					DescribeTable("should send a message to the user chanel", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
						// Arrange
						expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
						switch d := obj.(type) {
						case *v1beta1.Disruption:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
						case *v1beta1.DisruptionCron:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
						}

						By("not sending the message to the mirror slack channel")
						slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

						expectedSlackUser := &slack.User{
							ID:   uuid.New().String(),
							Name: expectedUserInfo.Username,
						}
						slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

						By("sending the message to the slack user")
						slackNotifMock.EXPECT().PostMessage(
							expectedSlackUser.ID,
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
						).Return("", "", nil).Once()

						// Act
						err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					},
						Entry("a success notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationSuccess),
						Entry("a completion notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationCompletion),
						Entry("a unknown notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationUnknown),
						Entry("a warning notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationWarning),
						Entry("an error notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationError),
						Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationInfo),
						Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationCompletion),
						Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationUnknown),
						Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationWarning),
						Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationError),
					)

					When("slack client fails to get the user by email", func() {
						DescribeTable("should not failed", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
							// Arrange
							expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
							switch d := obj.(type) {
							case *v1beta1.Disruption:
								Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							case *v1beta1.DisruptionCron:
								Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							}

							slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(nil, fmt.Errorf("could not get the user by email")).Once()

							By("not sending any message")
							slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage")

							// Act
							err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

							// Assert
							Expect(err).ShouldNot(HaveOccurred())
						},
							Entry("a success notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
								types.NotificationSuccess),
							Entry("a completion notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationCompletion),

							Entry("a unknown notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationUnknown),

							Entry("a warning notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationWarning),

							Entry("an error notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								}, types.NotificationError),
							Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationInfo),
							Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationCompletion),
							Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationUnknown),
							Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationWarning),
							Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationError),
						)
					})

					Context("with a slack configuration defined in the resource", func() {
						DescribeTable("should send a slack message the user chanel and the custom channel", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
							// Arrange
							expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}

							expectedReportingSlackChannel := "custom-slack-channel"
							expectedReporting := &v1beta1.Reporting{
								SlackChannel: expectedReportingSlackChannel,
							}
							switch d := obj.(type) {
							case *v1beta1.Disruption:
								obj.(*v1beta1.Disruption).Spec.Reporting = expectedReporting
								Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							case *v1beta1.DisruptionCron:
								obj.(*v1beta1.DisruptionCron).Spec.Reporting = expectedReporting
								Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							}

							By("not sending the message to the mirror slack channel")
							slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

							By("sending the message to the slack chanel from the reporting configuration")
							slackNotifMock.EXPECT().PostMessage(
								expectedReportingSlackChannel,
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
							).Return("", "", nil).Once()

							expectedSlackUser := &slack.User{
								ID:   uuid.New().String(),
								Name: expectedUserInfo.Username,
							}
							slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

							By("sending the message to the slack user")
							slackNotifMock.EXPECT().PostMessage(
								expectedSlackUser.ID,
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
								mock.AnythingOfType("slack.MsgOption"),
							).Return("", "", nil).Once()

							// Act
							err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

							// Assert
							Expect(err).ShouldNot(HaveOccurred())
						},
							Entry("a success notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
								types.NotificationSuccess,
							),
							Entry("a completion notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
								types.NotificationCompletion,
							),

							Entry("a unknown notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
								types.NotificationUnknown,
							),

							Entry("a warning notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
								types.NotificationWarning,
							),

							Entry("an error notification with a disruption",
								&v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
								types.NotificationError,
							),
							Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationInfo),
							Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationCompletion),
							Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationUnknown),
							Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationWarning),
							Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
								TypeMeta: metav1.TypeMeta{
									Kind: v1beta1.DisruptionCronKind,
								},
							}, types.NotificationError),
						)

						When("the message to the custom slack channel fails", func() {
							DescribeTable("should send a slack message the user chanel and the custom channel, but not returns an error", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
								// Arrange
								expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
								switch d := obj.(type) {
								case *v1beta1.Disruption:
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								case *v1beta1.DisruptionCron:
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								}

								expectedReportingSlackChannel := "custom-slack-channel"
								expectedReporting := &v1beta1.Reporting{
									SlackChannel: expectedReportingSlackChannel,
								}
								switch d := obj.(type) {
								case *v1beta1.Disruption:
									d.Spec.Reporting = expectedReporting
									obj = d
								case *v1beta1.DisruptionCron:
									d.Spec.Reporting = expectedReporting
									obj = d
								}

								By("not sending the message to the mirror slack channel")
								slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

								By("sending the message to the slack chanel from the reporting configuration, but it returns an error")
								slackNotifMock.EXPECT().PostMessage(
									expectedReportingSlackChannel,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", fmt.Errorf("could not send error the the reporting channel")).Once()

								expectedSlackUser := &slack.User{
									ID:   uuid.New().String(),
									Name: expectedUserInfo.Username,
								}
								slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

								By("sending the message to the slack user")
								slackNotifMock.EXPECT().PostMessage(
									expectedSlackUser.ID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								// Act
								err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

								// Assert
								Expect(err).ShouldNot(HaveOccurred())
							},
								Entry("a success notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
									types.NotificationSuccess,
								),
								Entry("a completion notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
									types.NotificationCompletion,
								),
								Entry("a unknown notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
									types.NotificationUnknown,
								),
								Entry("a warning notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
									types.NotificationWarning,
								),
								Entry("an error notification with a disruption", &v1beta1.Disruption{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionKind,
									},
								},
									types.NotificationError,
								),
								Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationInfo),
								Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationError),
							)

						})

						When("the userInfo username is not configured but SlackUserEmail is configured ", func() {
							DescribeTable("should send a slack message the user chanel", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {

								// Arrange
								expectedSlackUserName := "jaden@test.com"
								expectedReporting := &v1beta1.Reporting{
									SlackUserEmail: expectedSlackUserName,
								}

								switch d := obj.(type) {
								case *v1beta1.Disruption:
									d.Spec.Reporting = expectedReporting
								case *v1beta1.DisruptionCron:
									d.Spec.Reporting = expectedReporting
								}

								By("not sending the message to the mirror slack channel")
								slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

								expectedSlackUser := &slack.User{
									ID:   uuid.New().String(),
									Name: expectedSlackUserName,
								}
								slackNotifMock.EXPECT().GetUserByEmail(expectedSlackUserName).Return(expectedSlackUser, nil).Once()

								By("sending the message to the slack user")
								slackNotifMock.EXPECT().PostMessage(
									expectedSlackUser.ID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								// Act
								err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

								// Assert
								Expect(err).ShouldNot(HaveOccurred())
							},
								Entry("a success notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationSuccess,
								),
								Entry("a completion notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationCompletion,
								),

								Entry("a unknown notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationUnknown,
								),

								Entry("a warning notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationWarning,
								),

								Entry("an error notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationError,
								),
								Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationInfo),
								Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationError),
							)
						})

						When("the UserInfo username and SlackUserEmail are configured but differ in value", func() {
							DescribeTable("should prioritize utilizing the slackUserEmail", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
								// Arrange

								expectedUserInfo := v1.UserInfo{Username: "differingEmail@test.com"}
								expectedSlackUserName := "jaden@test.com"
								expectedReportingSlackChannel := "custom-slack-channel"
								expectedReporting := &v1beta1.Reporting{
									SlackChannel:   expectedReportingSlackChannel,
									SlackUserEmail: expectedSlackUserName,
								}

								switch d := obj.(type) {
								case *v1beta1.Disruption:
									d.Spec.Reporting = expectedReporting
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								case *v1beta1.DisruptionCron:
									d.Spec.Reporting = expectedReporting
									Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
								}

								By("not sending the message to the mirror slack channel")
								slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

								By("sending the message to the slack chanel from the reporting configuration")
								slackNotifMock.EXPECT().PostMessage(
									expectedReportingSlackChannel,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								expectedSlackUser := &slack.User{
									ID:   uuid.New().String(),
									Name: expectedSlackUserName,
								}
								slackNotifMock.EXPECT().GetUserByEmail(expectedSlackUserName).Return(expectedSlackUser, nil).Once()

								By("sending the message to the slack user")
								slackNotifMock.EXPECT().PostMessage(
									expectedSlackUser.ID,
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
									mock.AnythingOfType("slack.MsgOption"),
								).Return("", "", nil).Once()

								// Act
								err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

								// Assert
								Expect(err).ShouldNot(HaveOccurred())
							},
								Entry("a success notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationSuccess,
								),
								Entry("a completion notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationCompletion,
								),

								Entry("a unknown notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationUnknown,
								),

								Entry("a warning notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationWarning,
								),

								Entry("an error notification with a disruption",
									&v1beta1.Disruption{
										TypeMeta: metav1.TypeMeta{
											Kind: v1beta1.DisruptionKind,
										},
									},
									types.NotificationError,
								),
								Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationInfo),
								Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationCompletion),
								Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationUnknown),
								Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationWarning),
								Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
									TypeMeta: metav1.TypeMeta{
										Kind: v1beta1.DisruptionCronKind,
									},
								}, types.NotificationError),
							)
						})
					})
				})
			})

			When("the name of the user is not a valid address mail", func() {
				Context("with the slack configuration not in the resource and the invalid user is from userInfo", func() {
					DescribeTable("it should not return the error", func(ctx SpecContext, obj k8sclient.Object) {

						// Arrange
						expectedUserInfo := v1.UserInfo{Username: "not-an-valid-email"}
						switch d := obj.(type) {
						case *v1beta1.Disruption:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
						case *v1beta1.DisruptionCron:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
						}

						By("not sending the message to the mirror slack channel")
						slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

						// Act
						err := notifier.Notify(ctx, obj, corev1.Event{}, types.NotificationWarning)

						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					},
						Entry("a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}),
						Entry("a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}),
					)
				})

				Context("with the slack configuration in the resource and slackUserEmail is not valid", func() {
					DescribeTable("it should fallback to the userInfo user", func(ctx SpecContext, obj k8sclient.Object, notifType types.NotificationType) {
						expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}

						expectedReporting := &v1beta1.Reporting{
							SlackUserEmail: "invalid",
						}

						switch d := obj.(type) {
						case *v1beta1.Disruption:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							d.Spec.Reporting = expectedReporting
						case *v1beta1.DisruptionCron:
							Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
							d.Spec.Reporting = expectedReporting
						}

						By("not sending the message to the mirror slack channel")
						slackNotifMock.AssertNotCalled(GinkgoT(), "PostMessage", defaultMirrorSlackChanelID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

						expectedSlackUser := &slack.User{
							ID:   uuid.New().String(),
							Name: expectedUserInfo.Username,
						}
						slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

						By("sending the message to the user")
						slackNotifMock.EXPECT().PostMessage(
							expectedSlackUser.ID,
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
							mock.AnythingOfType("slack.MsgOption"),
						).Return("", "", nil).Once()

						//Act
						err := notifier.Notify(ctx, obj, corev1.Event{}, notifType)

						Expect(err).ShouldNot(HaveOccurred())

					},
						Entry("a success notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationSuccess),
						Entry("a completion notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationCompletion),
						Entry("a unknown notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationUnknown),
						Entry("a warning notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationWarning),
						Entry("an error notification with a disruption", &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}, types.NotificationError),
						Entry("a success notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationInfo),
						Entry("a completion notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationCompletion),
						Entry("a unknown notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationUnknown),
						Entry("a warning notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationWarning),
						Entry("an error notification with a disruptionCron", &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}, types.NotificationError),
					)

				})
			})
		})

		Describe("error cases", func() {
			When("the slack client fails to send a message to the user slack channel", func() {
				DescribeTable("it should return the error", func(ctx SpecContext, obj k8sclient.Object) {
					// Arrange
					expectedUserInfo := v1.UserInfo{Username: defaultUserInfoUserName}
					switch d := obj.(type) {
					case *v1beta1.Disruption:
						Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
					case *v1beta1.DisruptionCron:
						Expect(d.SetUserInfo(expectedUserInfo)).To(Succeed())
					}

					expectedSlackUser := &slack.User{
						ID:   uuid.New().String(),
						Name: expectedUserInfo.Username,
					}
					slackNotifMock.EXPECT().GetUserByEmail(expectedUserInfo.Username).Return(expectedSlackUser, nil).Once()

					By("sending the message to the slack chanel from the reporting configuration, but it returns an error")
					slackNotifMock.EXPECT().PostMessage(
						expectedSlackUser.ID,
						mock.AnythingOfType("slack.MsgOption"),
						mock.AnythingOfType("slack.MsgOption"),
						mock.AnythingOfType("slack.MsgOption"),
						mock.AnythingOfType("slack.MsgOption"),
						mock.AnythingOfType("slack.MsgOption"),
					).Return("", "", fmt.Errorf("could not send error the the reporting channel")).Once()

					// Act
					err := notifier.Notify(ctx, obj, corev1.Event{}, types.NotificationWarning)

					// Assert
					Expect(err).Should(HaveOccurred())
					Expect(err.Error()).Should(ContainSubstring("slack notifier: could not send error the the reporting channel"))
				},
					Entry("a disruption", &v1beta1.Disruption{
						TypeMeta: metav1.TypeMeta{
							Kind: v1beta1.DisruptionKind,
						},
					}),
					Entry("a disruptionCron", &v1beta1.DisruptionCron{
						TypeMeta: metav1.TypeMeta{
							Kind: v1beta1.DisruptionCronKind,
						},
					}),
				)
			})
		})
	})
})

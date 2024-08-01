// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package v1beta1

import (
	"go.uber.org/zap/zaptest"
	authV1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DisruptionCron Webhook", func() {
	var (
		defaultUserGroups    map[string]struct{}
		defaultUserGroupsStr string
	)

	BeforeEach(func() {
		// Arrange
		disruptionCronWebhookLogger = zaptest.NewLogger(GinkgoT()).Sugar()
		defaultUserGroups = map[string]struct{}{
			"group1": {},
			"group2": {},
		}
		defaultUserGroupsStr = "group1, group2"
	})

	AfterEach(func() {
		// Cleanup
		disruptionCronWebhookLogger = nil
		disruptionCronWebhookDeleteOnly = false
		disruptionCronPermittedUserGroups = nil
		defaultUserGroups = nil
		defaultUserGroupsStr = ""
	})

	Describe("ValidateCreate", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = false
				})

				It("should allow the creation", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("permitted user groups is present", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronPermittedUserGroups = defaultUserGroups
					disruptionCronPermittedUserGroupString = defaultUserGroupsStr
				})

				When("the userinfo is in the permitted user groups", func() {
					It("should allow the creation", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()
						Expect(disruptionCron.SetUserInfo(authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						})).To(Succeed())

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})
		})

		Describe("error cases", func() {
			When("the controller is in delete-only mode", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true
				})

				It("should not allow the creation", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					// Act
					warnings, err := disruptionCron.ValidateCreate()

					// Assert
					Expect(warnings).To(BeNil())

					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("the controller is currently in delete-only mode, you can't create new disruption cron for now"))
				})
			})

			When("permitted user groups is present", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronPermittedUserGroups = defaultUserGroups
					disruptionCronPermittedUserGroupString = defaultUserGroupsStr
				})

				When("the userinfo is not present", func() {
					It("should not allow the creation", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("user info not found in annotations")))
					})
				})

				When("the userinfo is not in the permitted user groups", func() {
					It("should not allow the creation", func() {
						// Arrange
						disruptionCron := makeValidDisruptionCron()
						Expect(disruptionCron.SetUserInfo(authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group3"},
						})).To(Succeed())

						// Act
						warnings, err := disruptionCron.ValidateCreate()

						// Assert
						Expect(warnings).To(BeNil())
						Expect(err).Should(HaveOccurred())
						Expect(err).To(MatchError(ContainSubstring("lacking sufficient authorization to create DisruptionCron. your user groups are group3, but you must be in one of the following groups: group1, group2")))
					})
				})
			})
		})
	})

	Describe("ValidateUpdate", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = false
				})

				It("should allow the update", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("the controller is in delete-only mode", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true
				})

				It("should allow the update", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					// Act
					warnings, err := disruptionCron.ValidateUpdate(makeValidDisruptionCron())

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("the user info has not changed", func() {
				It("should allow the update", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					userInfo := authV1.UserInfo{
						Username: "username@mail.com",
						Groups:   []string{"group1"},
					}
					Expect(disruptionCron.SetUserInfo(userInfo)).To(Succeed())

					oldDisruptionCron := makeValidDisruptionCron()
					Expect(oldDisruptionCron.SetUserInfo(userInfo)).To(Succeed())

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})

			})
		})

		Describe("error cases", func() {
			When("the user info has changed", func() {
				DescribeTable("should not allow the update", func(userInfo, oldUserInfo authV1.UserInfo) {
					// Arrange
					disruptionCron := makeValidDisruptionCron()
					Expect(disruptionCron.SetUserInfo(userInfo)).To(Succeed())

					oldDisruptionCron := makeValidDisruptionCron()
					Expect(oldDisruptionCron.SetUserInfo(oldUserInfo)).To(Succeed())

					// Act
					warnings, err := disruptionCron.ValidateUpdate(oldDisruptionCron)

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("the user info annotation is immutable")))
				},
					Entry("when the username has changed",
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						},
						authV1.UserInfo{
							Username: "differentusername@mail.com",
							Groups:   []string{"group1"},
						},
					),
					Entry("when the groups have changed",
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						},
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group2"},
						},
					),
					Entry("when the username and groups have changed",
						authV1.UserInfo{
							Username: "username@mail.com",
							Groups:   []string{"group1"},
						},
						authV1.UserInfo{
							Username: "newusername@mail.com",
							Groups:   []string{"group2"},
						},
					),
				)
			})
		})
	})

	Describe("ValidateDelete", func() {

		Describe("success cases", func() {
			When("the controller is not in delete-only mode", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = false
				})

				It("should allow the deletion", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					// Act
					warnings, err := disruptionCron.ValidateDelete()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			When("the controller is in delete-only mode", func() {

				BeforeEach(func() {
					// Arrange
					disruptionCronWebhookDeleteOnly = true
				})

				It("should allow the deletion", func() {
					// Arrange
					disruptionCron := makeValidDisruptionCron()

					// Act
					warnings, err := disruptionCron.ValidateDelete()

					// Assert
					Expect(warnings).To(BeNil())
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})
	})

})

func makeValidDisruptionCron() *DisruptionCron {
	return &DisruptionCron{
		TypeMeta: metav1.TypeMeta{
			Kind: DisruptionCronKind,
		},
	}
}

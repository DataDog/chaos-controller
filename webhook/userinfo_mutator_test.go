// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package webhook_test

import (
	"context"

	"net/http"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/webhook"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	v1 "k8s.io/api/admission/v1"
	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("UserInfoMutator", func() {
	var (
		logger *zap.SugaredLogger
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT()).Sugar()
	})

	Describe("Mutate", func() {
		var (
			decoder         admission.Decoder
			mockClient      *mocks.K8SClientMock
			userInfoMutator webhook.UserInfoMutator
		)

		BeforeEach(func() {
			mockClient = mocks.NewK8SClientMock(GinkgoT())
			decoder = admission.NewDecoder(runtime.NewScheme())
			userInfoMutator = webhook.UserInfoMutator{
				Client:  mockClient,
				Log:     logger,
				Decoder: decoder,
			}
		})

		Describe("success cases", func() {
			Context("when the request has user info", func() {
				DescribeTable("it should mutate the request", func(kind string) {
					// Arrange
					var object k8sclient.Object

					switch kind {
					case v1beta1.DisruptionKind:
						object = &v1beta1.Disruption{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionKind,
							},
						}
					case v1beta1.DisruptionCronKind:
						object = &v1beta1.DisruptionCron{
							TypeMeta: metav1.TypeMeta{
								Kind: v1beta1.DisruptionCronKind,
							},
						}
					}

					objectRaw, err := json.Marshal(object)
					Expect(err).ToNot(HaveOccurred())

					request := admission.Request{
						AdmissionRequest: v1.AdmissionRequest{
							Kind: metav1.GroupVersionKind{
								Kind: kind,
							},
							UserInfo: authv1.UserInfo{
								Username: "username@mail.com",
							},
							Object: runtime.RawExtension{
								Raw:    objectRaw,
								Object: object,
							},
						},
					}

					// Act
					response := userInfoMutator.Handle(context.TODO(), request)

					// Assert
					Expect(response.Allowed).To(BeTrue())
					Expect(*response.PatchType).To(Equal(v1.PatchTypeJSONPatch))
					Expect(response.Patches).To(ContainElement(
						MatchFields(IgnoreExtras, Fields{
							"Operation": Equal("add"),
							"Path":      Equal("/metadata/annotations"),
							"Value":     ContainElement(ContainSubstring("{\"username\":\"username@mail.com\"}")),
						}),
					))
				},
					Entry("with a disruption kind", v1beta1.DisruptionKind),
					Entry("with a disruption cron kind", v1beta1.DisruptionCronKind),
				)
			})
		})

		Describe("error cases", func() {
			When("the object kind is not valid", func() {
				DescribeTable("it should return an error", func(kind string) {
					// Arrange
					request := admission.Request{
						AdmissionRequest: v1.AdmissionRequest{
							Kind: metav1.GroupVersionKind{
								Kind: kind,
							},
						},
					}

					// Act
					response := userInfoMutator.Handle(context.TODO(), request)

					// Assert
					Expect(response.Allowed).To(BeFalse())
					Expect(response.Result.Code).To(Equal(int32(http.StatusBadRequest)))
					Expect(response.Result.Message).To(ContainSubstring("not a valid kind"))
				},
					Entry("with an invalid kind", "invalid-kind"),
					Entry("with an empty kind", ""),
					Entry("with a nil kind", nil),
					Entry("with a pod kind", "Pod"),
					Entry("with a deployment kind", "Deployment"),
					Entry("with a service kind", "Service"),
					Entry("with a namespace kind", "Namespace"),
					Entry("with a node kind", "Node"),
					Entry("with a configmap kind", "ConfigMap"),
				)
			})

			When("the decoder is nil", func() {
				DescribeTable("it should return an error", func(kind string) {
					// Arrange
					userInfoMutator.Decoder = nil
					request := admission.Request{
						AdmissionRequest: v1.AdmissionRequest{
							Kind: metav1.GroupVersionKind{
								Kind: kind,
							},
						},
					}

					// Act
					response := userInfoMutator.Handle(context.TODO(), request)

					// Assert
					Expect(response.Allowed).To(BeFalse())
					Expect(response.Result.Code).To(Equal(int32(http.StatusInternalServerError)))
					Expect(response.Result.Message).To(ContainSubstring("webhook Decoder seems to be nil while it should not"))

				},
					Entry("with a disruption kind", v1beta1.DisruptionKind),
					Entry("with a disruption cron kind", v1beta1.DisruptionCronKind),
				)
			})
		})
	})
})

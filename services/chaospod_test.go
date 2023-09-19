// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package services_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	chaosapi "github.com/DataDog/chaos-controller/api"
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	builderstest "github.com/DataDog/chaos-controller/builderstest"
	"github.com/DataDog/chaos-controller/cloudservice"
	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/o11y/metrics"
	"github.com/DataDog/chaos-controller/services"
	"github.com/DataDog/chaos-controller/targetselector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	DefaultNamespace                      = "namespace"
	DefaultChaosNamespace                 = "chaos-namespace"
	DefaultDisruptionName                 = "name"
	DefaultTargetName                     = "lorem"
	DefaultTargetNodeName                 = "ipsum"
	DefaultTargetPodIp                    = "10.10.10.10"
	DefaultHostPathDirectory              = v1.HostPathDirectory
	DefaultPathFile                       = v1.HostPathFile
	DefaultImagePullSecrets               = "pull-secret"
	DefaultInjectorServiceAccount         = "lorem"
	DefaultInjectorImage                  = "image"
	DefaultInjectorDNSDisruptionDNSServer = "8.8.8.8"
	DefaultInjectorDNSDisruptionKubeDNS   = "9.9.9.9"
	DefaultMetricsSinkName                = "name"
)

var _ = Describe("Chaos Pod Service", func() {

	var (
		chaosPod                          v1.Pod
		disruption                        *chaosv1beta1.Disruption
		k8sClientMock                     *mocks.K8SClientMock
		metricsSinkMock                   *metrics.SinkMock
		cloudServicesProvidersManagerMock *cloudservice.CloudServicesProvidersManagerMock
		targetSelectorMock                *targetselector.TargetSelectorMock
		chaosPodServiceConfig             services.ChaosPodServiceConfig
		chaosPodService                   services.ChaosPodService
		err                               error
		chaosPods                         []v1.Pod
	)

	BeforeEach(func() {
		// Arrange
		k8sClientMock = mocks.NewK8SClientMock(GinkgoT())
		targetSelectorMock = targetselector.NewTargetSelectorMock(GinkgoT())
		metricsSinkMock = metrics.NewSinkMock(GinkgoT())
		cloudServicesProvidersManagerMock = cloudservice.NewCloudServicesProvidersManagerMock(GinkgoT())
		disruption = &chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      DefaultDisruptionName,
				Namespace: DefaultNamespace,
			},
		}
		chaosPodServiceConfig = services.ChaosPodServiceConfig{}
	})

	JustBeforeEach(func() {
		// Arrange
		chaosPodServiceConfig.Log = logger
		chaosPodServiceConfig.ChaosNamespace = DefaultChaosNamespace
		chaosPodServiceConfig.MetricsSink = metricsSinkMock
		chaosPodServiceConfig.TargetSelector = targetSelectorMock
		chaosPodServiceConfig.CloudServicesProvidersManager = cloudServicesProvidersManagerMock
		if chaosPodServiceConfig.Client == nil {
			chaosPodServiceConfig.Client = k8sClientMock
		}

		// Action
		chaosPodService, err = services.NewChaosPodService(chaosPodServiceConfig)
		Expect(err).ShouldNot(HaveOccurred())
	})

	Describe("NewChaosPodService", func() {
		Context("with valid inputs", func() {
			It("should return a valid service", func() {
				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("return a valid chaosPodService")
				Expect(chaosPodService).ShouldNot(BeNil())
			})
		})

		Context("with a nil k8s client", func() {
			It("should return an error", func() {
				// Arrange
				chaosPodServiceConfig.Client = nil

				// Action
				chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)

				// Assert
				By("return an error")
				Expect(err).Should(HaveOccurred())
				Expect(err).To(MatchError("you must provide a non nil Kubernetes client"))

				By("not return a chaos pod service")
				Expect(chaosPodService).To(BeNil())
			})
		})
	})

	Describe("GetChaosPodsOfDisruption", func() {

		var (
			labelSets labels.Set
		)

		BeforeEach(func() {
			// Arrange
			labelSets = labels.Set{}
		})

		JustBeforeEach(func() {
			// Action
			chaosPods, err = chaosPodService.GetChaosPodsOfDisruption(context.Background(), disruption, labelSets)
		})

		Context("with three pods", func() {

			var (
				firstChaosPod, secondChaosPod, nonChaosPod v1.Pod
				nonChaosPodName                            = "pod-3"
				chaosPodsObjects                           = []client.Object{
					&firstChaosPod,
					&secondChaosPod,
					&nonChaosPod,
				}
				fakeClient client.Client
			)

			BeforeEach(func() {
				// Arrange
				firstChaosPod = builderstest.NewPodBuilder("pod-1", DefaultChaosNamespace).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").Build()
				secondChaosPod = builderstest.NewPodBuilder("pod-2", DefaultChaosNamespace).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").Build()
				nonChaosPod = builderstest.NewPodBuilder(nonChaosPodName, DefaultChaosNamespace).Build()

				fakeClient = fake.NewClientBuilder().WithObjects(chaosPodsObjects...).Build()
				chaosPodServiceConfig.Client = fakeClient
			})

			DescribeTable("success cases", func(ls labels.Set) {
				// Arrange
				chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)
				Expect(err).ShouldNot(HaveOccurred())

				// Action
				chaosPods, err := chaosPodService.GetChaosPodsOfDisruption(context.Background(), disruption, ls)

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("return a list of two pods")
				Expect(chaosPods).ToNot(BeEmpty())
				Expect(chaosPods).Should(HaveLen(2))

				for _, chaosPod := range chaosPods {
					Expect(chaosPod).ToNot(Equal(nonChaosPodName))
					Expect(chaosPod.Namespace).Should(Equal(DefaultChaosNamespace))
					Expect(chaosPod.Labels[chaostypes.DisruptionNameLabel]).Should(Equal(DefaultDisruptionName))
					Expect(chaosPod.Labels[chaostypes.DisruptionNamespaceLabel]).Should(Equal(DefaultNamespace))
				}
			},
				Entry("with an empty label set",
					labels.Set{},
				),
				Entry("with a nil label set",
					nil,
				),
			)

			Context("with a nil disruption and an empty label set", func() {

				BeforeEach(func() {
					// Arrange
					disruption = nil
					labelSets = labels.Set{}
				})

				Describe("success cases", func() {
					It("should return a list of all chaos pods", func() {
						// Assert
						By("not return an error")
						Expect(err).ShouldNot(HaveOccurred())

						By("return a list of two pods")
						Expect(chaosPods).ToNot(BeEmpty())
						Expect(chaosPods).Should(HaveLen(len(chaosPodsObjects)))
					})
				})
			})
		})

		Describe("failed cases", func() {
			When("the k8s client return an error", func() {

				BeforeEach(func() {
					// Arrange
					k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("error"))
				})

				It("should propagate the error", func() {
					// Assert
					By("return the error")
					Expect(err).Should(HaveOccurred())
					Expect(err).To(MatchError("error listing owned pods: error"))

					By("return an empty list of chaos pods")
					Expect(chaosPods).To(BeEmpty())
				})
			})
		})
	})

	Describe("HandleChaosPodTermination", func() {

		var (
			isFinalizerRemoved bool
			cpBuilder          *builderstest.ChaosPodBuilder
		)

		BeforeEach(func() {
			// Arrange
			cpBuilder = builderstest.NewPodBuilder("test-1", DefaultChaosNamespace)
			targetSelectorMock.EXPECT().TargetIsHealthy(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
		})

		JustBeforeEach(func() {
			// Arrange
			chaosPod = cpBuilder.Build()

			// Action
			isFinalizerRemoved, err = chaosPodService.HandleChaosPodTermination(context.Background(), disruption, &chaosPod)
		})

		DescribeTable("success cases", func(chaosPodBuilder *builderstest.ChaosPodBuilder) {
			// Arrange
			chaosPod := chaosPodBuilder.WithDeletion().WithChaosFinalizer().Build()
			target := chaosPod.Labels[chaostypes.TargetLabel]

			By("update the chaos pod object without the finalizer")
			k8sClientMock.EXPECT().Update(mock.Anything, &chaosPod).Return(nil)

			By("check if the TargetIsHealthy")
			targetSelectorMock.ExpectedCalls = nil
			targetSelectorMock.EXPECT().TargetIsHealthy(target, k8sClientMock, disruption).Return(nil)

			chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)
			Expect(err).ShouldNot(HaveOccurred())

			// Action
			isRemoved, err := chaosPodService.HandleChaosPodTermination(context.Background(), disruption, &chaosPod)

			// Assert
			By("not return an error")
			Expect(err).ShouldNot(HaveOccurred())

			By("remove the finalizer")
			Expect(chaosPod.GetFinalizers()).Should(Equal([]string{}))
			Expect(isRemoved).To(BeTrue())
		},
			Entry(
				"with a success pod",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithStatusPhase(v1.PodSucceeded)),
			Entry(
				"with a pending pod",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithStatusPhase(v1.PodPending)),
			Entry(
				"with a failed pod",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithStatusPhase(v1.PodFailed)),
			Entry(
				"with failed a pod exceeding its activeDeadlineSeconds",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithStatus(v1.PodStatus{
					Phase:             v1.PodFailed,
					Reason:            "DeadlineExceeded",
					ContainerStatuses: []v1.ContainerStatus{{}},
				})),
			Entry(
				"with a failed pod and an container injector in error state",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithStatus(v1.PodStatus{
					Phase: v1.PodFailed,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "injector",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "StartError",
								},
							},
						},
					},
				})),
			Entry(
				"with pod and an container injector in error state",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithStatus(v1.PodStatus{
					Phase: v1.PodFailed,
					ContainerStatuses: []v1.ContainerStatus{
						{
							Name: "injector",
							State: v1.ContainerState{
								Terminated: &v1.ContainerStateTerminated{
									Reason: "StartError",
								},
							},
						},
					},
				})),
			Entry(
				"with node failure running chaos pod",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", chaostypes.DisruptionKindNodeFailure).WithStatusPhase(v1.PodRunning)),
			Entry(
				"with container failure running chaos pod",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", chaostypes.DisruptionKindContainerFailure).WithStatusPhase(v1.PodRunning)),
		)

		DescribeTable("failures", func(chaosPod v1.Pod) {
			// Arrange
			target := chaosPod.Labels[chaostypes.TargetLabel]

			By("check if the TargetIsHealthy")
			targetSelectorMock.EXPECT().TargetIsHealthy(target, k8sClientMock, disruption).Return(nil)

			chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)
			Expect(err).ShouldNot(HaveOccurred())

			// Action
			isRemoved, err := chaosPodService.HandleChaosPodTermination(context.Background(), disruption, &chaosPod)

			// Assert
			By("not return an error")
			Expect(err).ShouldNot(HaveOccurred())

			By("not update the chaos pod object")
			k8sClientMock.AssertNotCalled(GinkgoT(), "Update")

			By("not remove the finalizer")
			Expect(chaosPod.GetFinalizers()).Should(Equal([]string{chaostypes.ChaosPodFinalizer}))
			Expect(isRemoved).To(BeFalse())
		},
			Entry("with a running pod",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").WithDeletion().WithChaosFinalizer().WithStatusPhase(v1.PodRunning).Build()),
			Entry("with a failed pod with containers",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").WithDeletion().WithChaosFinalizer().WithStatusPhase(
					v1.PodFailed,
				).WithContainerStatuses([]v1.ContainerStatus{{Name: "test-1"}}).Build()),
			Entry("with a failed pod with containers and a running injector",
				builderstest.NewPodBuilder(
					"test",
					DefaultNamespace,
				).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").WithDeletion().WithChaosFinalizer().WithStatusPhase(
					v1.PodFailed,
				).WithContainerStatuses([]v1.ContainerStatus{{Name: "injector"}}).Build()),
		)

		Context("with a chaos pod ready to be deleted", func() {

			BeforeEach(func() {
				// Arrange
				cpBuilder.WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").WithDeletion().WithChaosFinalizer()
			})

			Describe("success cases", func() {
				DescribeTable("when the TargetIsHealthy return an allowed error", func(targetErrStatus metav1.Status) {
					// Arrange
					By("check if the target is healthy")
					errorStatus := errors.StatusError{ErrStatus: targetErrStatus}

					targetSelectorMock.ExpectedCalls = nil
					targetSelectorMock.EXPECT().TargetIsHealthy(mock.Anything, mock.Anything, mock.Anything).Return(&errorStatus)

					By("update the chaos pod object without the finalizer")
					k8sClientMock.EXPECT().Update(mock.Anything, &chaosPod).Return(nil)

					chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)
					Expect(err).ShouldNot(HaveOccurred())

					// Action
					isRemoved, err := chaosPodService.HandleChaosPodTermination(context.Background(), disruption, &chaosPod)

					// Assert
					By("not return an error")
					Expect(err).ShouldNot(HaveOccurred())

					By("remove the finalizer")
					Expect(chaosPod.GetFinalizers()).Should(Equal([]string{}))
					Expect(isRemoved).To(BeTrue())
				},
					Entry("not found target", metav1.Status{
						Message: "Not found",
						Reason:  metav1.StatusReasonNotFound,
						Code:    http.StatusNotFound,
					}),
					Entry("pod is not running", metav1.Status{
						Message: "pod is not running",
					}),
					Entry("node is not ready", metav1.Status{
						Message: "node is not ready",
					}))
			})

			Describe("error cases", func() {
				When("the target is not healthy return an unexpected error", func() {

					BeforeEach(func() {
						// Arrange
						targetSelectorMock.ExpectedCalls = nil
						targetSelectorMock.EXPECT().TargetIsHealthy(mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("an error happend"))
					})

					It("should not remove the finalizer", func() {
						// Assert
						By("return an error")
						Expect(err).Should(HaveOccurred())

						By("not update the chaos pod object")
						k8sClientMock.AssertNotCalled(GinkgoT(), "Update")

						By("not remove the finalizer")
						Expect(chaosPod.GetFinalizers()).Should(Equal([]string{chaostypes.ChaosPodFinalizer}))
						Expect(isFinalizerRemoved).To(BeFalse())
					})

				})

				When("when the removeFinalizerForChaosPod return an error", func() {

					BeforeEach(func() {
						// Arrange
						errorStatus := errors.StatusError{ErrStatus: metav1.Status{
							Message: "node is not ready",
						}}
						targetSelectorMock.ExpectedCalls = nil
						targetSelectorMock.EXPECT().TargetIsHealthy(mock.Anything, mock.Anything, mock.Anything).Return(&errorStatus)

						k8sClientMock.EXPECT().Update(mock.Anything, mock.Anything).Return(fmt.Errorf("an error happened"))
					})

					It("should not remove the finalizer", func() {
						// Assert
						By("return an error")
						Expect(err).Should(HaveOccurred())

						By("not remove the finalizer")
						Expect(isFinalizerRemoved).To(BeFalse())
					})
				})

			})

			Context("with a succeeded pod", func() {

				BeforeEach(func() {
					// Arrange
					cpBuilder.WithStatusPhase(v1.PodSucceeded)
				})

				When("the k8s client return an error during the update", func() {

					BeforeEach(func() {
						// Arrange
						k8sClientMock.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("could not update"))
					})

					It("should propagate the error", func() {
						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(isFinalizerRemoved).To(BeFalse())
					})
				})

			})

			Context("with a running node failure chaos pod ", func() {
				BeforeEach(func() {
					// Arrange
					cpBuilder.WithStatusPhase(v1.PodRunning).WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", chaostypes.DisruptionKindNodeFailure)
				})

				When("the k8s client return an error during the update", func() {

					BeforeEach(func() {
						// Arrange
						k8sClientMock.EXPECT().Update(mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("could not update"))
					})

					It("should propagate the error", func() {
						// Assert
						Expect(err).Should(HaveOccurred())
						Expect(isFinalizerRemoved).To(BeFalse())
					})
				})
			})
		})

		Context("with a chaos pod not being deleted", func() {

			BeforeEach(func() {
				// Arrange
				cpBuilder.WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").WithChaosFinalizer()
			})

			It("should not remove the finalizer", func() {
				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("not remove the finalizer")
				k8sClientMock.AssertNotCalled(GinkgoT(), "Update", mock.Anything, mock.Anything)
				Expect(isFinalizerRemoved).To(BeFalse())
			})
		})

		Context("with a chaos pod without finalizer", func() {

			BeforeEach(func() {
				// Arrange
				cpBuilder.WithChaosPodLabels(DefaultDisruptionName, DefaultNamespace, "", "").WithDeletion().Build()
			})

			It("should not remove the finalizer", func() {
				// Assert
				By("not returning an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("not try to remove finalizer because it is already deleted")
				k8sClientMock.AssertNotCalled(GinkgoT(), "Update", mock.Anything, mock.Anything)

				By("return true because the finalizer is already removed")
				Expect(isFinalizerRemoved).To(BeTrue())
			})
		})
	})

	Describe("DeletePod", func() {

		var (
			pod       v1.Pod
			isDeleted bool
		)

		JustBeforeEach(func() {
			// Action
			isDeleted = chaosPodService.DeletePod(context.Background(), pod)
		})

		Context("with a pod not marked to be deleted", func() {

			BeforeEach(func() {
				// Arrange
				pod = builderstest.NewPodBuilder("test", DefaultNamespace).Build()
			})

			Describe("success cases", func() {

				Context("nominal case", func() {
					BeforeEach(func() {
						// Arrange
						By("tell the k8s client to delete the pod")
						k8sClientMock.EXPECT().Delete(mock.Anything, &pod).Return(nil)
					})

					It("should return true", func() {
						Expect(isDeleted).To(BeTrue())
					})
				})

				When("the k8s client return a not found error", func() {

					BeforeEach(func() {
						// Arrange
						errorNotFound := errors.StatusError{
							ErrStatus: metav1.Status{
								Message: "Not found",
								Reason:  metav1.StatusReasonNotFound,
								Code:    http.StatusNotFound,
							},
						}
						k8sClientMock.EXPECT().Delete(mock.Anything, &pod).Return(&errorNotFound)
					})

					It("should return true", func() {
						// Assert
						Expect(isDeleted).To(BeTrue())
					})
				})
			})

			Describe("error cases", func() {
				When("the k8s client return an error during the delete", func() {

					BeforeEach(func() {
						// Arrange
						k8sClientMock.EXPECT().Delete(mock.Anything, &pod).Return(fmt.Errorf("an error happened"))
					})

					It("should return false", func() {
						// Assert
						Expect(isDeleted).To(BeFalse())
					})
				})
			})
		})
	})

	Describe("GenerateChaosPodsOfDisruption", func() {

		var (
			targetContainers                             map[string]string
			DefaultInjectorNetworkDisruptionAllowedHosts []string
			dBuilder                                     *builderstest.DisruptionBuilder
			args                                         chaosapi.DisruptionArgs
			expectedArgs                                 []string
			disruptionKindName                           chaostypes.DisruptionKindName
		)

		BeforeEach(func() {
			// Arrange
			dBuilder = builderstest.NewDisruptionBuilder()
			targetContainers = map[string]string{"test": "test"}
			DefaultInjectorNetworkDisruptionAllowedHosts = []string{"10.10.10.10", "11.11.11.11"}
			chaosPodServiceConfig.Injector = services.ChaosPodServiceInjectorConfig{
				NetworkDisruptionAllowedHosts: DefaultInjectorNetworkDisruptionAllowedHosts,
				DNSDisruptionDNSServer:        DefaultInjectorDNSDisruptionDNSServer,
				DNSDisruptionKubeDNS:          DefaultInjectorDNSDisruptionKubeDNS,
			}
			pulseActiveDuration, pulseDormantDuration, pulseInitialDelay := time.Duration(0), time.Duration(0), time.Duration(0)
			args = chaosapi.DisruptionArgs{
				Level:                disruption.Spec.Level,
				TargetContainers:     targetContainers,
				TargetName:           DefaultTargetName,
				TargetNodeName:       DefaultTargetNodeName,
				TargetPodIP:          DefaultTargetPodIp,
				DryRun:               disruption.Spec.DryRun,
				DisruptionName:       disruption.Name,
				DisruptionNamespace:  DefaultNamespace,
				OnInit:               disruption.Spec.OnInit,
				PulseInitialDelay:    pulseInitialDelay,
				PulseActiveDuration:  pulseActiveDuration,
				PulseDormantDuration: pulseDormantDuration,
				MetricsSink:          DefaultMetricsSinkName,
				AllowedHosts:         DefaultInjectorNetworkDisruptionAllowedHosts,
				DNSServer:            DefaultInjectorDNSDisruptionDNSServer,
				KubeDNS:              DefaultInjectorDNSDisruptionKubeDNS,
				ChaosNamespace:       DefaultChaosNamespace,
			}
			metricsSinkMock.EXPECT().GetSinkName().Return(DefaultMetricsSinkName).Maybe()
		})

		JustBeforeEach(func() {
			// Arrange
			if disruptionKindName == "" {
				return
			}

			disruption := dBuilder.WithDisruptionKind(disruptionKindName).WithNamespace(DefaultNamespace).Build()

			notInjectedBefore := disruption.TimeToInject()

			subSpec := disruption.Spec.DisruptionKindPicker(disruptionKindName)

			args.Kind = disruptionKindName
			args.Level = disruption.Spec.Level
			args.TargetContainers = targetContainers
			args.DryRun = disruption.Spec.DryRun
			args.DisruptionName = disruption.Name
			args.OnInit = disruption.Spec.OnInit
			args.NotInjectedBefore = notInjectedBefore

			expectedArgs = args.CreateCmdArgs(subSpec.GenerateArgs())
			expectedArgs = append(expectedArgs, "--deadline", time.Now().Add(time.Second*2).Add(disruption.RemainingDuration()).Format(time.RFC3339))

			// Action
			chaosPods, err = chaosPodService.GenerateChaosPodsOfDisruption(&disruption, DefaultTargetName, DefaultTargetNodeName, targetContainers, DefaultTargetPodIp)
		})

		Describe("success cases", func() {

			DescribeTable("success cases", func(disruption chaosv1beta1.Disruption, expectedNumberOfChaosPods int) {
				// Arrange
				metricsSinkMock.EXPECT().GetSinkName().Return(DefaultMetricsSinkName).Maybe()

				chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)
				Expect(err).ShouldNot(HaveOccurred())

				// Action
				chaosPods, err := chaosPodService.GenerateChaosPodsOfDisruption(&disruption, DefaultTargetName, DefaultTargetNodeName, targetContainers, DefaultTargetPodIp)

				// Assert
				By("not return an error")
				Expect(err).ShouldNot(HaveOccurred())

				By("return two pods")
				Expect(chaosPods).To(HaveLen(expectedNumberOfChaosPods))
			},
				Entry("disruption with two kinds",
					builderstest.NewDisruptionBuilder().WithDisruptionKind(
						chaostypes.DisruptionKindNodeFailure,
					).WithDisruptionKind(
						chaostypes.DisruptionKindDiskFailure,
					).Build(),
					2,
				), Entry("disruption with one kind",
					builderstest.NewDisruptionBuilder().WithDisruptionKind(
						chaostypes.DisruptionKindNodeFailure,
					).Build(),
					1,
				),
				Entry("without disruption", nil, 0),
			)

			Context("with a disk failure disruption", func() {

				BeforeEach(func() {
					// Arrange
					disruptionKindName = chaostypes.DisruptionKindDiskFailure
				})

				It("should succeed", func() {
					// Assert
					By("not return an error")
					Expect(err).ShouldNot(HaveOccurred())

					By("return only one pod")
					Expect(chaosPods).To(HaveLen(1))

					By("having the correct container arguments")
					Expect(chaosPods[0].Spec.Containers[0].Args).Should(Equal(expectedArgs))
				})
			})

			Context("with a network disruption with a DisableDefaultAllowedHosts", func() {
				BeforeEach(func() {
					// Arrange
					disruptionKindName = chaostypes.DisruptionKindNetworkDisruption

					dBuilder.WithNetworkDisableDefaultAllowedHosts(true)

					args.AllowedHosts = make([]string, 0)
				})

				It("should succeed", func() {
					// Assert
					By("not return an error")
					Expect(err).ShouldNot(HaveOccurred())

					By("return only one pod")
					Expect(chaosPods).To(HaveLen(1))

					By("having the correct container arguments")
					Expect(chaosPods[0].Spec.Containers[0].Args).Should(Equal(expectedArgs))
				})

				Context("with a network cloud spec", func() {

					var serviceName string

					BeforeEach(func() {
						// Arrange
						serviceName = "GCP"

						cloudSpec := &chaosv1beta1.NetworkDisruptionCloudSpec{
							GCPServiceList: &[]chaosv1beta1.NetworkDisruptionCloudServiceSpec{
								{
									ServiceName: serviceName,
									Protocol:    "TCP",
									Flow:        "ingress",
									ConnState:   "open",
								},
							},
						}

						dBuilder.WithNetworkDisruptionCloudSpec(cloudSpec)
					})

					Context("nominal cases", func() {

						BeforeEach(func() {
							// Arrange
							cloudServicesProvidersManagerMock.EXPECT().GetServicesIPRanges(
								cloudtypes.CloudProviderName(serviceName),
								[]string{serviceName},
							).Return(map[string][]string{
								serviceName: {
									"10.0.0.0-10.10.10.10",
								},
							}, nil).Once()
						})

						It("should succeed", func() {
							// Assert
							By("not return an error")
							Expect(err).ShouldNot(HaveOccurred())

							By("return only one pod")
							Expect(chaosPods).To(HaveLen(1))

							By("having the correct service cloud args")
							Expect(chaosPods[0].Spec.Containers[0].Args).Should(ContainElements("--hosts", "10.0.0.0-10.10.10.10;0;TCP;ingress;open"))
						})
					})

					When("the cloud manager return an error during the fetching of services ip ranges", func() {
						BeforeEach(func() {
							// Arrange
							cloudServicesProvidersManagerMock.EXPECT().GetServicesIPRanges(
								mock.Anything,
								mock.Anything,
							).Return(nil, fmt.Errorf("an error happened"))
						})

						It("should propagate the error", func() {
							Expect(err).Should(HaveOccurred())
						})
					})
				})

				Context("with a Pulse Spec", func() {

					BeforeEach(func() {
						// Arrange
						pulseActiveDuration, pulseDormantDuration, pulseInitialDelay := time.Duration(10), time.Duration(11), time.Duration(12)

						dBuilder.WithSpecPulse(&chaosv1beta1.DisruptionPulse{
							ActiveDuration:  chaosv1beta1.DisruptionDuration(pulseActiveDuration.String()),
							DormantDuration: chaosv1beta1.DisruptionDuration(pulseDormantDuration.String()),
							InitialDelay:    chaosv1beta1.DisruptionDuration(pulseInitialDelay.String()),
						})

						args.PulseActiveDuration = pulseActiveDuration
						args.PulseDormantDuration = pulseDormantDuration
						args.PulseInitialDelay = pulseInitialDelay
					})

					It("should succeed", func() {
						// Assert
						By("not return an error")
						Expect(err).ShouldNot(HaveOccurred())

						By("return only one pod")
						Expect(chaosPods).To(HaveLen(1))

						By("having the correct container arguments")
						Expect(chaosPods[0].Spec.Containers[0].Args).Should(Equal(expectedArgs))
					})
				})
			})
		})
	})

	Describe("GenerateChaosPodOfDisruption", func() {
		var (
			DefaultTerminationGracePeriod int64
			DefaultActiveDeadlineSeconds  int64
			DefaultExpectedArgs           []string
			DefaultInjectorAnnotation     map[string]string
			DefaultInjectorLabels         map[string]string
			EmptyInjectorLabels           map[string]string
		)

		BeforeEach(func() {
			// Arrange
			DefaultTerminationGracePeriod = int64(60)
			DefaultActiveDeadlineSeconds = int64(disruption.RemainingDuration().Seconds()) + 10
			DefaultExpectedArgs = []string{
				"toto",
				"--deadline", time.Now().Add(disruption.RemainingDuration()).Format(time.RFC3339),
			}
			DefaultInjectorAnnotation = map[string]string{
				"lorem": "ipsum",
			}
			DefaultInjectorLabels = map[string]string{
				"ipsum": "dolores",
			}
			EmptyInjectorLabels = map[string]string{}
		})

		DescribeTable("success cases", func(expectedPodBuilder *builderstest.ChaosPodBuilder, expectedLabels map[string]string) {
			// Arrange
			expectedChaosPod := expectedPodBuilder.WithChaosSpec(
				DefaultTargetNodeName,
				DefaultTerminationGracePeriod,
				DefaultActiveDeadlineSeconds,
				DefaultExpectedArgs,
				DefaultHostPathDirectory,
				DefaultPathFile,
				DefaultInjectorServiceAccount,
				DefaultInjectorImage,
			).Build()

			imagePullSecrets := ""
			if expectedChaosPod.Spec.ImagePullSecrets != nil {
				imagePullSecrets = DefaultImagePullSecrets
			}

			chaosPodServiceConfig.Injector = services.ChaosPodServiceInjectorConfig{
				ServiceAccount:   DefaultInjectorServiceAccount,
				Image:            DefaultInjectorImage,
				Annotations:      DefaultInjectorAnnotation,
				Labels:           DefaultInjectorLabels,
				ImagePullSecrets: imagePullSecrets,
			}

			chaosPodService, err := services.NewChaosPodService(chaosPodServiceConfig)
			Expect(err).ShouldNot(HaveOccurred())

			args := []string{"toto"}
			kind := chaostypes.DisruptionKindNames[0]

			// Action
			chaosPod := chaosPodService.GenerateChaosPodOfDisruption(disruption, DefaultTargetName, DefaultTargetNodeName, args, kind)

			// Arrange
			// Remove containers args to avoid error due to a time.Now() which can diverge and create false negative results.
			for key := range chaosPod.Spec.Containers {
				chaosPod.Spec.Containers[key].Args = nil
				expectedChaosPod.Spec.Containers[key].Args = nil
			}

			// Assert
			By("return the expected spec")
			Expect(chaosPod.Spec).Should(Equal(expectedChaosPod.Spec))

			By("return the correct object meta")
			Expect(chaosPod.ObjectMeta.GenerateName).Should(Equal(fmt.Sprintf("chaos-%s-", DefaultDisruptionName)))
			Expect(chaosPod.ObjectMeta.Namespace).Should(Equal(DefaultChaosNamespace))
			Expect(chaosPod.ObjectMeta.Annotations).Should(Equal(DefaultInjectorAnnotation))
			Expect(chaosPod.ObjectMeta.Labels[chaostypes.TargetLabel]).Should(Equal(DefaultTargetName))
			Expect(chaosPod.ObjectMeta.Labels[chaostypes.DisruptionKindLabel]).Should(Equal(string(kind)))
			Expect(chaosPod.ObjectMeta.Labels[chaostypes.DisruptionNameLabel]).Should(Equal(DefaultDisruptionName))
			Expect(chaosPod.ObjectMeta.Labels[chaostypes.DisruptionNamespaceLabel]).Should(Equal(DefaultNamespace))
			for name, value := range expectedLabels {
				Expect(chaosPod.ObjectMeta.Labels[name]).Should(Equal(value))
			}

			By("add the finalizer")
			Expect(controllerutil.ContainsFinalizer(&chaosPod, chaostypes.ChaosPodFinalizer)).To(BeTrue())
		},
			Entry("chaos pod without image pull secrets",
				builderstest.NewPodBuilder(
					"pod-1",
					DefaultChaosNamespace,
				),
				EmptyInjectorLabels),
			Entry("chaos pod with image pull secrets",
				builderstest.NewPodBuilder(
					"pod-1",
					DefaultChaosNamespace,
				).WithPullSecrets([]v1.LocalObjectReference{
					{
						Name: DefaultImagePullSecrets,
					},
				}),
				EmptyInjectorLabels),
			Entry("chaos pod with injector labels",
				builderstest.NewPodBuilder(
					"pod-1",
					DefaultChaosNamespace,
				).WithLabels(DefaultInjectorLabels),
				DefaultInjectorLabels),
		)
	})

	Describe("GetPodInjectorArgs", func() {

		var chaosPodArgs []string

		JustBeforeEach(func() {
			// Action
			chaosPodArgs = chaosPodService.GetPodInjectorArgs(chaosPod)
		})

		Describe("success cases", func() {
			Context("with a single chaos pod", func() {

				Context("with a single container with args", func() {
					BeforeEach(func() {
						// Arrange
						chaosPod = builderstest.NewPodBuilder("test-1", DefaultNamespace).WithChaosSpec(
							DefaultTargetNodeName,
							int64(60),
							int64(60),
							[]string{
								"1",
								"2",
							},
							DefaultHostPathDirectory,
							DefaultPathFile,
							DefaultInjectorServiceAccount,
							DefaultInjectorImage,
						).Build()
					})

					It("should return the chaos pod args", func() {
						// Assert
						Expect(chaosPodArgs).Should(Equal([]string{
							"1",
							"2",
						}))
					})
				})

				Context("without container", func() {
					BeforeEach(func() {
						// Arrange
						chaosPod = builderstest.NewPodBuilder("test-1", DefaultNamespace).Build()
					})

					It("should return an empty args", func() {
						// Assert
						Expect(chaosPodArgs).Should(Equal([]string{}))
					})
				})

			})
		})
	})

	Describe("CreatePod", func() {
		BeforeEach(func() {
			// Arrange
			chaosPod = builderstest.NewPodBuilder("test-1", DefaultNamespace).Build()
		})

		JustBeforeEach(func() {
			// Action
			err = chaosPodService.CreatePod(context.Background(), &chaosPod)
		})

		Describe("success case", func() {
			BeforeEach(func() {
				// Arrange
				By("create the chaos pod with the ks8 client")
				k8sClientMock.EXPECT().Create(mock.Anything, &chaosPod).Return(nil)
			})

			It("should not return an error", func() {
				// Assert
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Describe("error cases", func() {
			When("the k8s client return an error during the create", func() {

				BeforeEach(func() {
					// Arrange
					By("create the chaos pod with the ks8 client")
					k8sClientMock.EXPECT().Create(mock.Anything, &chaosPod).Return(fmt.Errorf("an error happened"))
				})

				It("should propagate the error", func() {
					// Assert
					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})

	Describe("WaitForPodCreation", func() {
		JustBeforeEach(func() {
			// Action
			err = chaosPodService.WaitForPodCreation(context.Background(), chaosPod)
		})

		Context("with a single pod", func() {

			BeforeEach(func() {
				// Arrange
				chaosPod = builderstest.NewPodBuilder("test-1", DefaultNamespace).Build()
			})

			Describe("success cases", func() {

				BeforeEach(func() {
					// Arrange
					By("call the Get method of the k8s client")
					errorStatus := errors.StatusError{
						ErrStatus: metav1.Status{
							Message: "Not found",
							Reason:  metav1.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						},
					}
					k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{Namespace: chaosPod.Namespace, Name: chaosPod.Name}, &chaosPod).Return(&errorStatus).Once()
					k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{Namespace: chaosPod.Namespace, Name: chaosPod.Name}, &chaosPod).Return(nil).Once()
				})

				It("should not return an error", func() {
					// Assert
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			Describe("error cases", func() {

				When("the Get method of the k8s client an error", func() {

					BeforeEach(func() {
						// Arrange
						By("call the Get method of the k8s client")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{Namespace: chaosPod.Namespace, Name: chaosPod.Name}, &chaosPod).Return(fmt.Errorf("")).Once()
					})

					It("should return an error", func() {
						// Assert
						Expect(err).Should(HaveOccurred())
					})
				})

			})
		})
	})

	Describe("HandleOrphanedChaosPods", func() {
		var (
			DefaultLs  map[string]string
			DefaultReq ctrl.Request
		)

		BeforeEach(func() {
			// Arrange
			metricsSinkMock.EXPECT().MetricOrphanFound(mock.Anything).Return(nil).Maybe()
			DefaultLs = map[string]string{
				chaostypes.DisruptionNameLabel:      DefaultDisruptionName,
				chaostypes.DisruptionNamespaceLabel: DefaultNamespace,
			}
			DefaultReq = ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: DefaultNamespace,
					Name:      DefaultDisruptionName,
				},
			}
		})

		JustBeforeEach(func() {
			// Action
			err = chaosPodService.HandleOrphanedChaosPods(context.Background(), DefaultReq)
		})

		Describe("success cases", func() {

			Context("with three chaos pods", func() {
				BeforeEach(func() {
					// Arrange
					chaosPods = []v1.Pod{
						builderstest.NewPodBuilder("test-1", DefaultNamespace).WithChaosFinalizer().WithChaosPodLabels(DefaultDisruptionName, DefaultDisruptionName, DefaultTargetName, chaostypes.DisruptionKindDiskFailure).Build(),
						builderstest.NewPodBuilder("test-2", DefaultNamespace).WithChaosFinalizer().WithChaosPodLabels(DefaultDisruptionName, DefaultDisruptionName, DefaultTargetName, chaostypes.DisruptionKindDiskFailure).Build(),
						builderstest.NewPodBuilder("test-3", DefaultNamespace).WithChaosFinalizer().WithChaosPodLabels(DefaultDisruptionName, DefaultDisruptionName, DefaultTargetName, chaostypes.DisruptionKindDiskFailure).Build(),
					}
				})

				Context("nominal cases", func() {

					BeforeEach(func() {
						// Arrange
						By("list the existing chaos pods matching criteria")
						k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
							Namespace:     DefaultChaosNamespace,
							LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
						}).Return(nil).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
							list.(*v1.PodList).Items = chaosPods
						}).Once()

						By("check if the target exist for each chaos pods")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{
							Namespace: DefaultNamespace,
							Name:      DefaultTargetName,
						}, mock.Anything).Return(&errors.StatusError{ErrStatus: metav1.Status{
							Message: "Not found",
							Reason:  metav1.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						}}).Times(3)

						for _, pod := range chaosPods {
							By("remove the finalizer of all chaos pods")
							podWithoutFinalizer := pod.DeepCopy()
							controllerutil.RemoveFinalizer(podWithoutFinalizer, chaostypes.ChaosPodFinalizer)
							k8sClientMock.EXPECT().Update(mock.Anything, podWithoutFinalizer).Return(nil).Once()

							By("remove all chaos pods")
							k8sClientMock.EXPECT().Delete(mock.Anything, podWithoutFinalizer).Return(nil).Once()
						}
					})

					It("should remove orphan chaos pods", func() {
						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})

				When("an error occur during the removing of the finalizer for all chaos pods", func() {

					BeforeEach(func() {
						// Arrange
						By("list the existing chaos pods matching criteria")
						k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
							Namespace:     DefaultChaosNamespace,
							LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
						}).Return(nil).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
							list.(*v1.PodList).Items = chaosPods
						}).Once()

						By("check if the target exist for each chaos pods")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{
							Namespace: DefaultNamespace,
							Name:      DefaultTargetName,
						}, mock.Anything).Return(&errors.StatusError{ErrStatus: metav1.Status{
							Message: "Not found",
							Reason:  metav1.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						}}).Times(3)

						k8sClientMock.EXPECT().Update(mock.Anything, mock.Anything).Return(fmt.Errorf("an error happened")).Times(3)
					})

					It("should not remove the chaos pod", func() {
						// Assert
						By("not return an error")
						Expect(err).ShouldNot(HaveOccurred())

						By("not delete chaos pods")
						k8sClientMock.AssertNotCalled(GinkgoT(), "Delete")
					})
				})

				When("an error occur during the removing of the finalizer for a single chaos pod", func() {

					BeforeEach(func() {
						// Arrange
						By("list the existing chaos pods matching criteria")
						k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
							Namespace:     DefaultChaosNamespace,
							LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
						}).Return(nil).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
							list.(*v1.PodList).Items = chaosPods
						}).Once()

						By("check if the target exist for each chaos pods")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{
							Namespace: DefaultNamespace,
							Name:      DefaultTargetName,
						}, mock.Anything).Return(&errors.StatusError{ErrStatus: metav1.Status{
							Message: "Not found",
							Reason:  metav1.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						}}).Times(3)

						for i, pod := range chaosPods {
							podWithoutFinalizer := pod.DeepCopy()
							controllerutil.RemoveFinalizer(podWithoutFinalizer, chaostypes.ChaosPodFinalizer)

							if i == 1 {
								By("return an error for the second chaos pod")
								k8sClientMock.EXPECT().Update(mock.Anything, podWithoutFinalizer).Return(fmt.Errorf("an error occured")).Once()

								continue
							}

							By("remove the finalizer of all chaos pods")
							k8sClientMock.EXPECT().Update(mock.Anything, podWithoutFinalizer).Return(nil).Once()

							By("remove all chaos pods")
							k8sClientMock.EXPECT().Delete(mock.Anything, podWithoutFinalizer).Return(nil).Once()
						}
					})

					It("should not remove the chaos pod", func() {
						// Assert
						Expect(err).ShouldNot(HaveOccurred())

						By("remove only two chaos pods")
						k8sClientMock.AssertNumberOfCalls(GinkgoT(), "Delete", 2)
					})
				})

				When("an error occur during the removing of the chaos pod for all chaos pods", func() {

					BeforeEach(func() {
						// Arrange
						By("list the existing chaos pods matching criteria")
						k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
							Namespace:     DefaultChaosNamespace,
							LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
						}).Return(nil).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
							list.(*v1.PodList).Items = chaosPods
						}).Once()

						By("check if the target exist for each chaos pods")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{
							Namespace: DefaultNamespace,
							Name:      DefaultTargetName,
						}, mock.Anything).Return(&errors.StatusError{ErrStatus: metav1.Status{
							Message: "Not found",
							Reason:  metav1.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						}}).Times(3)

						k8sClientMock.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Times(3)

						k8sClientMock.EXPECT().Delete(mock.Anything, mock.Anything).Return(fmt.Errorf("an error occured")).Times(3)
					})

					It("should not remove the chaos pod", func() {
						// Assert
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})

			Context("with a single chaos pod", func() {

				BeforeEach(func() {
					// Arrange
					chaosPods = []v1.Pod{
						builderstest.NewPodBuilder("test-1", DefaultNamespace).WithChaosFinalizer().WithChaosPodLabels(DefaultDisruptionName, DefaultDisruptionName, DefaultTargetName, chaostypes.DisruptionKindDiskFailure).Build(),
					}
				})

				Context("the target still exist", func() {

					BeforeEach(func() {
						// Arrange
						By("list the existing chaos pods matching criteria")
						k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
							Namespace:     DefaultChaosNamespace,
							LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
						}).Return(nil).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
							list.(*v1.PodList).Items = chaosPods
						}).Once()

						By("check if the target exist")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{
							Namespace: DefaultNamespace,
							Name:      DefaultTargetName,
						}, mock.Anything).Return(nil).Once()
					})

					It("should not remove the non orphan chaos pod", func() {
						// Assert
						Expect(err).ShouldNot(HaveOccurred())

						By("not remove the finalizer")
						k8sClientMock.AssertNotCalled(GinkgoT(), "Update")

						By("not delete the chaos pod")
						k8sClientMock.AssertNotCalled(GinkgoT(), "Delete")
					})
				})

				When("the k8s client return an unexpected error during the verification of the target", func() {

					BeforeEach(func() {
						// Arrange
						chaosPods := []v1.Pod{
							builderstest.NewPodBuilder("test-1", DefaultNamespace).WithChaosFinalizer().WithChaosPodLabels(DefaultDisruptionName, DefaultDisruptionName, DefaultTargetName, chaostypes.DisruptionKindDiskFailure).Build(),
						}

						By("list the existing chaos pods matching criteria")
						k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
							Namespace:     DefaultChaosNamespace,
							LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
						}).Return(nil).Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
							list.(*v1.PodList).Items = chaosPods
						}).Once()

						By("check if the target exist")
						k8sClientMock.EXPECT().Get(mock.Anything, types.NamespacedName{
							Namespace: DefaultNamespace,
							Name:      DefaultTargetName,
						}, mock.Anything).Return(fmt.Errorf("an error happened")).Once()
					})

					It("should not remove the non orphan chaos pods", func() {
						// Assert
						Expect(err).ShouldNot(HaveOccurred())

						By("not remove the finalizer")
						k8sClientMock.AssertNotCalled(GinkgoT(), "Update")

						By("not delete the chaos pod")
						k8sClientMock.AssertNotCalled(GinkgoT(), "Delete")
					})
				})
			})
		})

		Describe("error cases", func() {

			When("GetChaosPodsOfDisruption return an error", func() {

				BeforeEach(func() {
					// Arrange
					By("list the existing chaos pods matching criteria")
					k8sClientMock.EXPECT().List(mock.Anything, mock.Anything, &client.ListOptions{
						Namespace:     DefaultChaosNamespace,
						LabelSelector: labels.SelectorFromValidatedSet(DefaultLs),
					}).Return(fmt.Errorf("an error happened")).Once()
				})

				It("should propagate the error", func() {
					// Assert
					Expect(err).Should(HaveOccurred())

					By("not verify the presence of the target")
					k8sClientMock.AssertNotCalled(GinkgoT(), "Get")

					By("not remove the finalizer")
					k8sClientMock.AssertNotCalled(GinkgoT(), "Update")

					By("not delete the chaos pod")
					k8sClientMock.AssertNotCalled(GinkgoT(), "Delete")
				})
			})
		})
	})
})

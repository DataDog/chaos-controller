// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package targetselector_test

import (
	"context"
	"os"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/mocks"
	"github.com/DataDog/chaos-controller/targetselector"
	"github.com/DataDog/chaos-controller/types"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	// ChaosFailureInjectionImageVariableName is the name of the chaos failure injection image variable
	ChaosFailureInjectionImageVariableName = "CHAOS_INJECTOR_IMAGE"
)

var (
	runningPod1 *corev1.Pod
	runningPod2 *corev1.Pod
	failedPod   *corev1.Pod
	pendingPod  *corev1.Pod
)

var (
	mixedStatusPods []corev1.Pod
)

var (
	runningNode *corev1.Node
	failedNode  *corev1.Node
)

var (
	justRunningNodes []corev1.Node
)

var _ = Describe("Helpers", func() {
	var k8SClientMock *mocks.K8SClientMock
	var image string
	var disruption *chaosv1beta1.Disruption
	var targetSelector targetselector.TargetSelector

	BeforeEach(func() {
		targetSelector = targetselector.NewRunningTargetSelector(false, "foo")

		k8SClientMock = mocks.NewK8SClientMock(GinkgoT())

		runningPod1 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "runningPod",
				Namespace: "bar",
			},
			Spec: corev1.PodSpec{
				NodeName: "runningNode",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "foo",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		runningPod2 = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anotherRunningPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "foo",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		failedPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failedPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		}

		pendingPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pendingPod",
				Namespace: "bar",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				InitContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "chaos-handler",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		mixedStatusPods = []corev1.Pod{
			*runningPod1,
			*runningPod2,
			*failedPod,
			*pendingPod,
		}

		runningNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "runningNode",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		failedNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "failedNode",
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionFalse,
					},
				},
			},
		}

		justRunningNodes = []corev1.Node{
			*runningNode,
		}

		image = "chaos-injector:latest"
		os.Setenv(ChaosFailureInjectionImageVariableName, image)

		disruption = &chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Selector: map[string]string{"foo": "bar"},
				NodeFailure: &chaosv1beta1.NodeFailureSpec{
					Shutdown: false,
				},
				ContainerFailure: &chaosv1beta1.ContainerFailureSpec{
					Forced: false,
				},
				Network: &chaosv1beta1.NetworkDisruptionSpec{
					Hosts: []chaosv1beta1.NetworkDisruptionHostSpec{
						{
							Host:     "127.0.0.1",
							Port:     80,
							Protocol: "tcp",
						},
					},
					Drop:           0,
					Corrupt:        0,
					Delay:          1000,
					BandwidthLimit: 10000,
				},
				CPUPressure: &chaosv1beta1.CPUPressureSpec{},
				DiskPressure: &chaosv1beta1.DiskPressureSpec{
					Path:       "/mnt/foo",
					Throttling: chaosv1beta1.DiskPressureThrottlingSpec{},
				},
			},
		}
	})

	AfterEach(func() {
		os.Unsetenv(ChaosFailureInjectionImageVariableName)
	})

	Describe("GetMatchingPodsOverTotalPods", func() {
		Context("with empty label selector", func() {
			It("should return an error", func() {
				// Arrange
				disruption.Namespace = ""
				disruption.Spec.Selector = nil

				// Act & Assert
				Expect(targetSelector.GetMatchingPodsOverTotalPods(nil, disruption)).Error().To(HaveOccurred())
			})
		})

		Context("with non-empty label selector", func() {
			BeforeEach(func() {
				disruption.Namespace = "foo"
				disruption.Spec.Selector = map[string]string{
					"app": "bar",
				}
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"bar"},
					},
				}
			})

			It("should pass given selector for the given namespace to the client", func() {
				// Arrange
				var capturedNamespace string
				var capturedLabelSelector string
				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.PodList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						for _, opt := range opts {
							if listOpts, ok := opt.(*client.ListOptions); ok {
								capturedNamespace = listOpts.Namespace
								if listOpts.LabelSelector != nil {
									capturedLabelSelector = listOpts.LabelSelector.String()
								}
							}
						}
						if podList, ok := list.(*corev1.PodList); ok {
							podList.Items = mixedStatusPods
						}
					}).Return(nil)

				// Act
				_, _, err := targetSelector.GetMatchingPodsOverTotalPods(k8SClientMock, disruption)

				// Assert
				Expect(err).ToNot(HaveOccurred())
				Expect(capturedNamespace).To(Equal("foo"))
				Expect(capturedLabelSelector).To(Equal("app=bar,app,!app,app in (bar),app notin (bar)"))
			})

			It("should return the pods list except for failed pod", func() {
				// Arrange
				disruption.Namespace = ""

				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.PodList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						if podList, ok := list.(*corev1.PodList); ok {
							podList.Items = mixedStatusPods
						}
					}).Return(nil)

				// Act
				r, _, err := targetSelector.GetMatchingPodsOverTotalPods(k8SClientMock, disruption)
				numExcludedPods := 2 // pending + failed pods

				// Assert
				Expect(err).ToNot(HaveOccurred())
				Expect(r.Items).To(HaveLen(len(mixedStatusPods) - numExcludedPods))
			})
		})

		Context("with on init mode enabled", func() {
			BeforeEach(func() {
				disruption.Spec.OnInit = true
			})

			It("should match pending pods with init containers only", func() {
				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.PodList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						if podList, ok := list.(*corev1.PodList); ok {
							podList.Items = mixedStatusPods
						}
					}).Return(nil)

				r, _, err := targetSelector.GetMatchingPodsOverTotalPods(k8SClientMock, disruption)
				Expect(err).ToNot(HaveOccurred())
				Expect(r.Items[0]).To(Equal(*pendingPod))
			})
		})

		Context("with controller safeguards enabled", func() {
			BeforeEach(func() {
				targetSelector = targetselector.NewRunningTargetSelector(true, "runningNode")
			})

			It("should exclude the pods running on the same node as the controller from targets", func() {
				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.PodList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						if podList, ok := list.(*corev1.PodList); ok {
							podList.Items = mixedStatusPods
						}
					}).Return(nil)

				r, _, err := targetSelector.GetMatchingPodsOverTotalPods(k8SClientMock, disruption)

				Expect(err).ToNot(HaveOccurred())
				Expect(r.Items).To(HaveLen(1)) // only the pod not running on the same node as the controller
			})
		})
	})

	Describe("GetMatchingNodesOverTotalNodes", func() {
		Context("with empty label selector", func() {
			It("should return an error", func() {
				// Arrange
				disruption.Spec.Selector = nil

				// Act
				_, _, err := targetSelector.GetMatchingNodesOverTotalNodes(k8SClientMock, disruption)

				// Assert
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with non-empty label selector", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{
					"app": "bar",
				}
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpDoesNotExist,
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"bar"},
					},
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpNotIn,
						Values:   []string{"bar"},
					},
				}
			})

			It("should pass given selector to the client", func() {
				var capturedLabelSelector string
				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.NodeList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						for _, opt := range opts {
							if listOpts, ok := opt.(*client.ListOptions); ok {
								if listOpts.LabelSelector != nil {
									capturedLabelSelector = listOpts.LabelSelector.String()
								}
							}
						}
						if nodeList, ok := list.(*corev1.NodeList); ok {
							nodeList.Items = justRunningNodes
						}
					}).Return(nil)

				_, _, err := targetSelector.GetMatchingNodesOverTotalNodes(k8SClientMock, disruption)
				Expect(err).ToNot(HaveOccurred())
				Expect(capturedLabelSelector).To(Equal("app=bar,app,!app,app in (bar),app notin (bar)"))
			})

			It("should return the nodes list with no error", func() {
				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.NodeList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						if nodeList, ok := list.(*corev1.NodeList); ok {
							nodeList.Items = justRunningNodes
						}
					}).Return(nil)

				r, _, err := targetSelector.GetMatchingNodesOverTotalNodes(k8SClientMock, disruption)

				Expect(err).ToNot(HaveOccurred())
				Expect(r.Items).To(HaveLen(len(justRunningNodes)))
				Expect(r.Items[0].Name).To(Equal("runningNode"))
			})
		})

		Context("with controller safeguards enabled", func() {
			BeforeEach(func() {
				targetSelector = targetselector.NewRunningTargetSelector(true, "runningNode")
			})

			It("should exclude the controller node from targets", func() {
				k8SClientMock.EXPECT().List(mock.Anything, mock.AnythingOfType("*v1.NodeList"), mock.Anything).
					Run(func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) {
						if nodeList, ok := list.(*corev1.NodeList); ok {
							nodeList.Items = justRunningNodes
						}
					}).Return(nil)

				r, _, err := targetSelector.GetMatchingNodesOverTotalNodes(k8SClientMock, disruption)

				Expect(err).ToNot(HaveOccurred())
				Expect(r.Items).To(BeEmpty())
			})
		})
	})

	Describe("TargetIsHealthy", func() {
		Context("with pod-level disruption spec", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{"foo": "bar"}
				disruption.Spec.Level = types.DisruptionLevelPod
			})

			It("should return no error for running pod", func() {
				// Arrange
				k8SClientMock.EXPECT().
					Get(mock.Anything, k8stypes.NamespacedName{Name: "runningPod", Namespace: "default"}, mock.AnythingOfType("*v1.Pod")).
					Run(func(ctx context.Context, key k8stypes.NamespacedName, obj client.Object, opts ...client.GetOption) {
						if pod, ok := obj.(*corev1.Pod); ok {
							*pod = *runningPod1
						}
					}).
					Return(nil)

				// Since the disruption has NodeFailure spec, it will also check the pod's node
				k8SClientMock.EXPECT().
					Get(mock.Anything, k8stypes.NamespacedName{Name: "runningNode", Namespace: ""}, mock.AnythingOfType("*v1.Node")).
					Run(func(ctx context.Context, key k8stypes.NamespacedName, obj client.Object, opts ...client.GetOption) {
						if node, ok := obj.(*corev1.Node); ok {
							*node = *runningNode
						}
					}).
					Return(nil)

				// Act
				err := targetSelector.TargetIsHealthy("runningPod", k8SClientMock, disruption)

				// Assert
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return error for failed pod", func() {
				// Arrange
				k8SClientMock.EXPECT().
					Get(mock.Anything, k8stypes.NamespacedName{Name: "failedPod", Namespace: "default"}, mock.AnythingOfType("*v1.Pod")).
					Run(func(ctx context.Context, key k8stypes.NamespacedName, obj client.Object, opts ...client.GetOption) {
						if pod, ok := obj.(*corev1.Pod); ok {
							*pod = *failedPod
						}
					}).Return(nil)

				// Act
				err := targetSelector.TargetIsHealthy("failedPod", k8SClientMock, disruption)

				// Assert
				Expect(err).To(HaveOccurred())
			})
		})

		Context("with node-level disruption spec", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{"foo": "bar"}
				disruption.Spec.Level = types.DisruptionLevelNode
			})

			It("should not return an error for running node", func() {
				// Arrange
				k8SClientMock.EXPECT().Get(mock.Anything, k8stypes.NamespacedName{Name: "runningNode", Namespace: ""}, mock.AnythingOfType("*v1.Node")).
					Run(func(ctx context.Context, key k8stypes.NamespacedName, obj client.Object, opts ...client.GetOption) {
						if node, ok := obj.(*corev1.Node); ok {
							*node = *runningNode
						}
					}).Return(nil)

				// Act
				err := targetSelector.TargetIsHealthy("runningNode", k8SClientMock, disruption)

				// Assert
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error for failed node", func() {
				k8SClientMock.EXPECT().Get(mock.Anything, k8stypes.NamespacedName{Name: "failedNode", Namespace: ""}, mock.AnythingOfType("*v1.Node")).
					Run(func(ctx context.Context, key k8stypes.NamespacedName, obj client.Object, opts ...client.GetOption) {
						if node, ok := obj.(*corev1.Node); ok {
							*node = *failedNode
						}
					}).Return(nil)

				err := targetSelector.TargetIsHealthy("failedNode", k8SClientMock, disruption)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetLabelSelectorFromInstance", func() {
		Context("with empty selectors", func() {
			It("should return an error when both selectors are nil", func() {
				// Arrange
				disruption.Spec.Selector = nil
				disruption.Spec.AdvancedSelector = nil

				// Act
				_, err := targetselector.GetLabelSelectorFromInstance(disruption)

				// Assert
				Expect(err).To(MatchError("selector can't be an empty set"))
			})

			It("should return an error when both selectors are empty", func() {
				// Arrange
				disruption.Spec.Selector = map[string]string{}
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{}

				// Act
				_, err := targetselector.GetLabelSelectorFromInstance(disruption)

				// Assert
				Expect(err).To(MatchError("selector can't be an empty set"))
			})
		})

		Context("with simple selector only", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{
					"app": "test",
					"env": "staging",
				}
				disruption.Spec.AdvancedSelector = nil
			})

			It("should create label selector from simple selector", func() {
				// Act
				selector, err := targetselector.GetLabelSelectorFromInstance(disruption)

				// Assert
				Expect(err).ToNot(HaveOccurred())
				Expect(selector).ToNot(BeNil())
				Expect(selector.String()).To(Equal("app=test,env=staging"))
			})
		})

		Context("with advanced selector only", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = nil
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "app",
						Operator: metav1.LabelSelectorOpExists,
					},
					{
						Key:      "env",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"staging", "prod"},
					},
				}
			})

			It("should create label selector from advanced selector", func() {
				selector, err := targetselector.GetLabelSelectorFromInstance(disruption)
				Expect(err).ToNot(HaveOccurred())
				Expect(selector).ToNot(BeNil())
				Expect(selector.String()).To(Equal("app,env in (prod,staging)"))
			})
		})

		Context("with both simple and advanced selectors", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{
					"app": "test",
				}
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "env",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"staging"},
					},
				}
			})

			It("should combine both selectors", func() {
				selector, err := targetselector.GetLabelSelectorFromInstance(disruption)
				Expect(err).ToNot(HaveOccurred())
				Expect(selector).ToNot(BeNil())
				Expect(selector.String()).To(Equal("app=test,env in (staging)"))
			})
		})

		Context("with OnInit enabled", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = map[string]string{
					"app": "test",
				}
				disruption.Spec.OnInit = true
			})

			It("should add the disrupt-on-init label requirement", func() {
				// Act
				selector, err := targetselector.GetLabelSelectorFromInstance(disruption)

				// Assert
				Expect(err).ToNot(HaveOccurred())
				Expect(selector).ToNot(BeNil())
				Expect(selector.String()).To(ContainSubstring("app=test"))
				Expect(selector.String()).To(ContainSubstring("chaos.datadoghq.com/disrupt-on-init"))
			})
		})

		Context("with invalid advanced selector", func() {
			BeforeEach(func() {
				disruption.Spec.Selector = nil
				disruption.Spec.AdvancedSelector = []metav1.LabelSelectorRequirement{
					{
						Key:      "invalid key with spaces",
						Operator: metav1.LabelSelectorOpExists,
					},
				}
			})

			It("should return an error for invalid advanced selector", func() {
				_, err := targetselector.GetLabelSelectorFromInstance(disruption)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package injector_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/DataDog/chaos-controller/injector"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("NodeReplacement", func() {
	var (
		config     NodeReplacementInjectorConfig
		k8sClient  kubernetes.Interface
		inj        Injector
		spec       v1beta1.NodeReplacementSpec
		targetNode *corev1.Node
		targetPod  *corev1.Pod
		targetPVC  *corev1.PersistentVolumeClaim
		nodeIP     = "192.168.1.10"
		podIP      = "10.244.0.5"
		nodeName   = "test-node"
		podName    = "test-pod"
		podUID     = "test-pod-uid-123"
		pvcName    = "test-pvc"
		namespace  = "test-namespace"
	)

	BeforeEach(func() {
		// Create test node
		targetNode = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: nodeName,
			},
			Spec: corev1.NodeSpec{
				Unschedulable: false,
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: nodeIP,
					},
				},
			},
		}

		// Create test pod
		targetPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: namespace,
				UID:       types.UID(podUID),
			},
			Spec: corev1.PodSpec{
				NodeName: nodeName,
				Volumes: []corev1.Volume{
					{
						Name: "test-volume",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: pvcName,
							},
						},
					},
				},
			},
			Status: corev1.PodStatus{
				PodIP: podIP,
				Phase: corev1.PodRunning,
			},
		}

		// Create test PVC
		targetPVC = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: namespace,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
			},
		}

		// Create fake Kubernetes client with test resources
		k8sClient = fake.NewSimpleClientset(targetNode, targetPod, targetPVC)

		config = NodeReplacementInjectorConfig{
			Config: Config{
				Log:         log,
				MetricsSink: ms,
				K8sClient:   k8sClient,
				Disruption: chaosapi.DisruptionArgs{
					TargetNodeName: nodeName,
					TargetPodIP:    podIP,
					Level:          chaostypes.DisruptionLevelNode,
					DryRun:         false,
				},
			},
		}

		spec = v1beta1.NodeReplacementSpec{
			DeleteStorage:      true,
			ForceDelete:        false,
			GracePeriodSeconds: nil,
		}
	})

	Describe("constructor validation", func() {
		Context("when target node name is missing", func() {
			BeforeEach(func() {
				config.Disruption.TargetNodeName = ""
			})

			It("should return an error", func() {
				_, err := NewNodeReplacementInjector(spec, config)
				Expect(err).To(MatchError("target node name is required"))
			})
		})
	})

	// Context for successful construction
	Context("with valid config", func() {
		JustBeforeEach(func() {
			var err error
			inj, err = NewNodeReplacementInjector(spec, config)
			Expect(err).ToNot(HaveOccurred())
			Expect(inj).ToNot(BeNil())
		})

	Describe("GetDisruptionKind", func() {
		It("should return the correct disruption kind", func() {
			Expected := chaostypes.DisruptionKindNodeReplacement
			Actual := inj.GetDisruptionKind()
			Expect(string(Actual)).To(Equal(string(Expected)))
		})
	})

	Describe("TargetName", func() {
		It("should return the target node name", func() {
			Expect(inj.TargetName()).To(Equal(nodeName))
		})
	})

	Describe("injection", func() {
		JustBeforeEach(func() {
			Expect(inj.Inject()).To(Succeed())
		})

		Context("with default settings", func() {
			It("should cordon the node", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Unschedulable).To(BeTrue())
			})

			It("should delete the PVC when deleteStorage is true", func() {
				_, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})

			It("should delete the target pod", func() {
				_, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("with deleteStorage disabled", func() {
			BeforeEach(func() {
				spec.DeleteStorage = false
			})

			It("should not delete the PVC", func() {
				pvc, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(pvc.Name).To(Equal(pvcName))
			})

			It("should still delete the target pod", func() {
				_, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("with force delete enabled", func() {
			BeforeEach(func() {
				spec.ForceDelete = true
			})

			It("should delete the pod with zero grace period", func() {
				// In a real cluster, this would use grace period 0, but fake client doesn't track this
				// We verify the pod is deleted, which indicates the delete call was made
				_, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("with custom grace period", func() {
			var gracePeriod int64 = 30

			BeforeEach(func() {
				spec.GracePeriodSeconds = &gracePeriod
			})

			It("should delete the pod with custom grace period", func() {
				// In a real cluster, this would use the custom grace period, but fake client doesn't track this
				// We verify the pod is deleted, which indicates the delete call was made
				_, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})

	Describe("idempotency", func() {
		Context("when the target pod no longer exists", func() {
			JustBeforeEach(func() {
				// First injection should work normally
				Expect(inj.Inject()).To(Succeed())

				// Create a new pod with same IP but different UID (simulating StatefulSet recreation)
				newPod := targetPod.DeepCopy()
				newPod.UID = types.UID("new-pod-uid-456")
				_, err := k8sClient.CoreV1().Pods(namespace).Create(context.Background(), newPod, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Second injection should be idempotent (not process the new pod with same IP)
				Expect(inj.Inject()).To(Succeed())
			})

			It("should not process the same pod UID twice", func() {
				// Verify the new pod still exists (wasn't processed again)
				pod, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(string(pod.UID)).To(Equal("new-pod-uid-456"))
			})
		})
	})

	Describe("error handling", func() {
		Context("when target pod IP is missing", func() {
			BeforeEach(func() {
				config.Disruption.TargetPodIP = ""
			})

			It("should return an error during injection", func() {
				err := inj.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("target pod IP is required"))
			})
		})

		Context("when target pod does not exist", func() {
			BeforeEach(func() {
				config.Disruption.TargetPodIP = "1.2.3.4" // Non-existent pod IP
			})

			It("should return an error during injection", func() {
				err := inj.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("target pod with IP 1.2.3.4 not found"))
			})
		})

		Context("when node does not exist", func() {
			BeforeEach(func() {
				// Delete the node to simulate missing node
				err := k8sClient.CoreV1().Nodes().Delete(context.Background(), nodeName, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("should return an error during injection", func() {
				err := inj.Inject()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to get node"))
			})
		})
	})

	Describe("dry run mode", func() {
		BeforeEach(func() {
			config.Disruption.DryRun = true
		})

		JustBeforeEach(func() {
			Expect(inj.Inject()).To(Succeed())
		})

		It("should not actually cordon the node", func() {
			node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Spec.Unschedulable).To(BeFalse())
		})

		It("should not delete the PVC", func() {
			pvc, err := k8sClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pvc.Name).To(Equal(pvcName))
		})

		It("should not delete the pod", func() {
			pod, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Name).To(Equal(podName))
		})
	})

	Describe("cleanup", func() {
		Context("when node was cordoned by this injector", func() {
			JustBeforeEach(func() {
				// Inject first to cordon the node
				Expect(inj.Inject()).To(Succeed())

				// Verify node is cordoned
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Unschedulable).To(BeTrue())

				// Now clean up
				Expect(inj.Clean()).To(Succeed())
			})

			It("should uncordon the node", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Unschedulable).To(BeFalse())
			})
		})

		Context("when node was already uncordoned", func() {
			BeforeEach(func() {
				// Ensure node starts uncordoned
				targetNode.Spec.Unschedulable = false
			})

			It("should not return an error", func() {
				Expect(inj.Clean()).To(Succeed())
			})
		})

		Context("in dry run mode", func() {
			BeforeEach(func() {
				config.Disruption.DryRun = true
			})

			JustBeforeEach(func() {
				// Inject and clean in dry run mode
				Expect(inj.Inject()).To(Succeed())
				Expect(inj.Clean()).To(Succeed())
			})

			It("should not modify the node", func() {
				node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(node.Spec.Unschedulable).To(BeFalse())
			})
		})
	})

	}) // End of "with valid config" context
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector_test

import (
	"context"

	chaosapi "github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
	chaostypes "github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Pod replacement target identity", func() {
	It("deletes only the selected name and namespace when host-network IPs are shared", func() {
		makePod := func(namespace, name string) *corev1.Pod {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
				Spec:       corev1.PodSpec{NodeName: "node", HostNetwork: true},
				Status:     corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "192.0.2.1"},
			}
		}
		node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node"}}
		bystander := makePod("a-system", "a-bystander")
		target := makePod("z-app", "z-target")
		client := fake.NewSimpleClientset(node, bystander, target)
		inj, err := injector.NewPodReplacementInjector(v1beta1.PodReplacementSpec{}, injector.PodReplacementInjectorConfig{
			Config: injector.Config{
				Log:       log,
				K8sClient: client,
				Disruption: chaosapi.DisruptionArgs{
					Level:               chaostypes.DisruptionLevelPod,
					TargetName:          target.Name,
					DisruptionNamespace: target.Namespace,
					TargetNodeName:      node.Name,
					TargetPodIP:         target.Status.PodIP,
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(inj.Inject()).To(Succeed())

		_, err = client.CoreV1().Pods(bystander.Namespace).Get(context.Background(), bystander.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred(), "the unselected pod must not be deleted")
		_, err = client.CoreV1().Pods(target.Namespace).Get(context.Background(), target.Name, metav1.GetOptions{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), "the selected pod must be deleted")
	})
})

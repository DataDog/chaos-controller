// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package controllers

import (
	"bytes"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	chaostypes "github.com/DataDog/chaos-controller/types"
)

var _ = Describe("CPU Pressure", func() {
	var (
		cpuStress chaosv1beta1.Disruption
		targetPod corev1.Pod
	)

	BeforeEach(func() {
		cpuStress = chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				Duration:   "1m",
				Count:      &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Containers: []string{"ctn1"},
				CPUPressure: &chaosv1beta1.CPUPressureSpec{
					Count: &intstr.IntOrString{Type: intstr.String, StrVal: "100%"},
				},
			},
		}
	})

	JustBeforeEach(func(ctx SpecContext) {
		Skip("See CHAOS-XXX: Data Race in test")
		cpuStress, targetPod, _ = InjectPodsAndDisruption(ctx, cpuStress, true)
	})

	DescribeTable("targeted container is stopped", func(ctx SpecContext, forced bool) {
		ExpectDisruptionStatus(ctx, cpuStress, chaostypes.DisruptionInjectionStatusInjected)

		stopTargetedContainer := chaosv1beta1.Disruption{
			ObjectMeta: metav1.ObjectMeta{
				Name:        targetPod.Name + "-term",
				Namespace:   namespace,
				Annotations: map[string]string{chaosv1beta1.SafemodeEnvironmentAnnotation: "lima"},
			},
			Spec: chaosv1beta1.DisruptionSpec{
				AllowDisruptedTargets: true,
				StaticTargeting:       true,
				Duration:              "15s",
				Count:                 &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				Containers:            []string{"ctn1"},
				ContainerFailure: &chaosv1beta1.ContainerFailureSpec{
					Forced: forced,
				},
			},
		}

		Concurrently{
			func(ctx SpecContext) {
				Consistently(func(ctx SpecContext) bool {
					return ExpectDisruptionStatus(ctx, cpuStress, chaostypes.DisruptionInjectionStatusInjected, chaostypes.DisruptionInjectionStatusPausedInjected, chaostypes.DisruptionInjectionStatusPreviouslyInjected)
				}).WithContext(ctx).Within(calcDisruptionGoneTimeout(cpuStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(BeTrue())
			},
			func(ctx SpecContext) {
				GinkgoHelper()

				CreateDisruption(ctx, stopTargetedContainer, targetPod)

				// First, we wait to have the container failure injected
				ExpectDisruptionStatus(ctx, stopTargetedContainer, chaostypes.DisruptionInjectionStatusInjected)

				// Then the container failure disruption should disappear so we stop killing the container we want to stress
				ExpectDisruptionStatus(ctx, stopTargetedContainer, chaostypes.DisruptionInjectionStatusPreviouslyInjected)

				// once it's done, we expect to eventually have stressers back before the end of the disruption
				Eventually(func(g Gomega, ctx SpecContext) {
					err := k8sClient.Get(ctx, types.NamespacedName{Namespace: cpuStress.Namespace, Name: cpuStress.Name}, &cpuStress)
					if apierrors.IsNotFound(err) {
						return
					}
					g.Expect(err).ToNot(HaveOccurred())

					// get chaos pod associated to original disruption, the cpu-pressure
					chaosPods, err := listChaosPods(ctx, cpuStress)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(chaosPods.Items).To(HaveLen(1))

					AddReportEntry("Gonna exec into pod", chaosPods.Items[0].Name)

					// count number of chaos-injector cpu-stress, there should be 2
					stou, sterr, err := ExecuteRemoteCommand(ctx, &chaosPods.Items[0], "injector", `ps ax | grep "/usr/local/bin/chaos-injector cpu-stress" | grep -v grep | wc -l`)
					g.Expect(err).ToNot(HaveOccurred(), "an unexpected error occured while executing remote command, details:", sterr)

					AddReportEntry("Found cpu-stress process on injector:", stou)

					g.Expect(stou).To(WithTransform(strings.TrimSpace, Equal("1")))
				}).WithContext(ctx).Within(calcDisruptionGoneTimeout(cpuStress)).ProbeEvery(disruptionPotentialChangesEvery).Should(Succeed())
			},
		}.DoAndWait(ctx)
	},
		Entry("by a SIGTERM", false),
		Entry("by a SIGKILL", true),
	)
})

func ExecuteRemoteCommand(ctx SpecContext, pod *v1.Pod, container, command string) (string, string, error) {
	coreClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return "", "", fmt.Errorf("unable to create kubernetes client: %w", err)
	}

	request := coreClient.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command:   []string{"/bin/sh", "-c", command},
			Container: container,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", request.URL())
	Expect(err).ToNot(HaveOccurred())

	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", "", fmt.Errorf("%w Failed executing command %s on %v/%v", err, command, pod.Namespace, pod.Name)
	}

	return buf.String(), errBuf.String(), nil
}

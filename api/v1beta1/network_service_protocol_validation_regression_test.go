// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Network service protocol admission", func() {
	DescribeTable("rejects only selected unsupported protocols", func(ports []NetworkDisruptionServicePortSpec, rejected bool) {
		scheme := runtime.NewScheme()
		Expect(v1.AddToScheme(scheme)).To(Succeed())
		service := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "mixed", Namespace: "default"},
			Spec: v1.ServiceSpec{
				Type: v1.ServiceTypeClusterIP, ClusterIP: "10.96.0.10",
				Ports: []v1.ServicePort{
					{Name: "tcp", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: v1.ProtocolTCP},
					{Name: "sctp", Port: 9090, TargetPort: intstr.FromInt(9090), Protocol: v1.ProtocolSCTP},
				},
			},
		}
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(service).Build()
		err := validateServices(client, []NetworkDisruptionServiceSpec{{Name: "mixed", Namespace: "default", Ports: ports}})
		if rejected {
			Expect(err).To(MatchError(ContainSubstring("SCTP")))
		} else {
			Expect(err).To(Succeed())
		}
	},
		Entry("all ports include SCTP", []NetworkDisruptionServicePortSpec(nil), true),
		Entry("SCTP selected by name", []NetworkDisruptionServicePortSpec{{Name: "sctp"}}, true),
		Entry("SCTP selected by number", []NetworkDisruptionServicePortSpec{{Port: 9090}}, true),
		Entry("TCP-only name selection", []NetworkDisruptionServicePortSpec{{Name: "tcp"}}, false),
		Entry("TCP-only number selection", []NetworkDisruptionServicePortSpec{{Port: 8080}}, false),
	)
})

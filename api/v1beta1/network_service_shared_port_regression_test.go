// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Network service ports sharing a number", func() {
	tcp := v1.ServicePort{Name: "dns-tcp", Port: 53, TargetPort: intstr.FromInt(5353), Protocol: v1.ProtocolTCP}
	udp := v1.ServicePort{Name: "dns-udp", Port: 53, TargetPort: intstr.FromInt(5353), Protocol: v1.ProtocolUDP}
	DescribeTable("preserves every matching service port", func(request v1beta1.NetworkDisruptionServicePortSpec, expected []v1.ServicePort) {
		for _, ports := range [][]v1.ServicePort{{tcp, udp}, {udp, tcp}} {
			service := &v1.Service{Spec: v1.ServiceSpec{Type: v1.ServiceTypeClusterIP, Ports: ports}}
			spec := v1beta1.NetworkDisruptionServiceSpec{Ports: []v1beta1.NetworkDisruptionServicePortSpec{request}}
			selected, missing := spec.ExtractAffectedPortsInServicePorts(service)
			Expect(missing).To(BeEmpty())
			Expect(selected).To(ConsistOf(expected))
		}
	},
		Entry("numeric selector matches TCP and UDP", v1beta1.NetworkDisruptionServicePortSpec{Port: 53}, []v1.ServicePort{tcp, udp}),
		Entry("name and number select TCP regardless of declaration order", v1beta1.NetworkDisruptionServicePortSpec{Name: "dns-tcp", Port: 53}, []v1.ServicePort{tcp}),
		Entry("name and number select UDP regardless of declaration order", v1beta1.NetworkDisruptionServicePortSpec{Name: "dns-udp", Port: 53}, []v1.ServicePort{udp}),
		Entry("name-only selection remains supported", v1beta1.NetworkDisruptionServicePortSpec{Name: "dns-tcp"}, []v1.ServicePort{tcp}),
	)
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package safemode_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/safemode"
)

var _ = Describe("AddAllSafemodeObjects", func() {
	var dis v1beta1.Disruption

	BeforeEach(func() {
		dis = v1beta1.Disruption{}
	})

	It("returns only Generic when no specs set", func() {
		result := safemode.AddAllSafemodeObjects(dis, nil)
		Expect(result).To(HaveLen(1))
		_, ok := result[0].(*safemode.Generic)
		Expect(ok).To(BeTrue())
	})

	DescribeTable("adds one extra element for each non-nil spec",
		func(setter func(*v1beta1.Disruption), expectedType interface{}) {
			setter(&dis)
			result := safemode.AddAllSafemodeObjects(dis, nil)
			Expect(result).To(HaveLen(2))
			_, ok := result[0].(*safemode.Generic)
			Expect(ok).To(BeTrue(), "first element must be Generic")
			Expect(result[1]).To(BeAssignableToTypeOf(expectedType))
		},
		Entry("Network", func(d *v1beta1.Disruption) { d.Spec.Network = &v1beta1.NetworkDisruptionSpec{} }, &safemode.Network{}),
		Entry("DiskPressure", func(d *v1beta1.Disruption) { d.Spec.DiskPressure = &v1beta1.DiskPressureSpec{} }, &safemode.DiskPressure{}),
		Entry("DiskFailure", func(d *v1beta1.Disruption) { d.Spec.DiskFailure = &v1beta1.DiskFailureSpec{} }, &safemode.DiskFailure{}),
		Entry("ContainerFailure", func(d *v1beta1.Disruption) { d.Spec.ContainerFailure = &v1beta1.ContainerFailureSpec{} }, &safemode.ContainerFailure{}),
		Entry("CPUPressure", func(d *v1beta1.Disruption) { d.Spec.CPUPressure = &v1beta1.CPUPressureSpec{} }, &safemode.CPU{}),
		Entry("MemoryPressure", func(d *v1beta1.Disruption) { d.Spec.MemoryPressure = &v1beta1.MemoryPressureSpec{} }, &safemode.Memory{}),
		Entry("GRPC", func(d *v1beta1.Disruption) { d.Spec.GRPC = &v1beta1.GRPCDisruptionSpec{} }, &safemode.GRPC{}),
		Entry("NodeFailure", func(d *v1beta1.Disruption) { d.Spec.NodeFailure = &v1beta1.NodeFailureSpec{} }, &safemode.Node{}),
		Entry("DNS", func(d *v1beta1.Disruption) { d.Spec.DNS = &v1beta1.DNSDisruptionSpec{} }, &safemode.DNS{}),
	)

	It("returns all 10 types when all specs are set", func() {
		dis.Spec.Network = &v1beta1.NetworkDisruptionSpec{}
		dis.Spec.DiskPressure = &v1beta1.DiskPressureSpec{}
		dis.Spec.DiskFailure = &v1beta1.DiskFailureSpec{}
		dis.Spec.ContainerFailure = &v1beta1.ContainerFailureSpec{}
		dis.Spec.CPUPressure = &v1beta1.CPUPressureSpec{}
		dis.Spec.MemoryPressure = &v1beta1.MemoryPressureSpec{}
		dis.Spec.GRPC = &v1beta1.GRPCDisruptionSpec{}
		dis.Spec.NodeFailure = &v1beta1.NodeFailureSpec{}
		dis.Spec.DNS = &v1beta1.DNSDisruptionSpec{}
		result := safemode.AddAllSafemodeObjects(dis, nil)
		Expect(result).To(HaveLen(10))
	})
})

var _ = Describe("Reinit", func() {
	It("calls Init on every safemode in the list", func() {
		dis := v1beta1.Disruption{}
		mock1 := safemode.NewSafemodeMock(GinkgoT())
		mock2 := safemode.NewSafemodeMock(GinkgoT())
		mock1.EXPECT().Init(mock.Anything, mock.Anything).Return()
		mock2.EXPECT().Init(mock.Anything, mock.Anything).Return()
		safemode.Reinit([]safemode.Safemode{mock1, mock2}, dis, nil)
	})

	It("is a no-op for empty list", func() {
		Expect(func() {
			safemode.Reinit([]safemode.Safemode{}, v1beta1.Disruption{}, nil)
		}).NotTo(Panic())
	})
})

var _ = Describe("Init methods", func() {
	var dis v1beta1.Disruption

	BeforeEach(func() {
		dis = v1beta1.Disruption{}
	})

	DescribeTable("stores disruption without panic",
		func(sm safemode.Safemode) {
			Expect(func() { sm.Init(dis, nil) }).NotTo(Panic())
		},
		Entry("Generic", &safemode.Generic{}),
		Entry("Network", &safemode.Network{}),
		Entry("CPU", &safemode.CPU{}),
		Entry("Memory", &safemode.Memory{}),
		Entry("DiskPressure", &safemode.DiskPressure{}),
		Entry("DiskFailure", &safemode.DiskFailure{}),
		Entry("ContainerFailure", &safemode.ContainerFailure{}),
		Entry("GRPC", &safemode.GRPC{}),
		Entry("Node", &safemode.Node{}),
		Entry("DNS", &safemode.DNS{}),
	)
})

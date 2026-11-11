// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1

import (
	"context"

	"github.com/DataDog/chaos-controller/cloudservice"
	cloudtypes "github.com/DataDog/chaos-controller/cloudservice/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

var _ = Describe("Cloud network filter safety", func() {
	BeforeEach(func() {
		savedManager, savedLogger := cloudServicesProvidersManager, logger
		logger = zap.NewNop().Sugar()
		DeferCleanup(func() {
			cloudServicesProvidersManager, logger = savedManager, savedLogger
		})
	})

	DescribeTable("checks resolved cloud filters without modifying the admitted spec", func(provider string, ranges []string, wantCaught bool) {
		services := []NetworkDisruptionCloudServiceSpec{{ServiceName: "service"}}
		cloud := &NetworkDisruptionCloudSpec{}
		switch provider {
		case "AWS":
			cloud.AWSServiceList = &services
		case "GCP":
			cloud.GCPServiceList = &services
		case "Datadog":
			cloud.DatadogServiceList = &services
		}
		disruption := &Disruption{Spec: DisruptionSpec{
			Level:      "pod",
			Network:    &NetworkDisruptionSpec{Drop: 100, Cloud: cloud},
			Unsafemode: &UnsafemodeSpec{DisableCountTooLarge: true},
		}}
		Expect(disruption.Spec.Network.Validate()).To(Succeed())
		manager := cloudservice.NewCloudServicesProvidersManagerMock(GinkgoT())
		cloudServicesProvidersManager = manager
		manager.EXPECT().GetServicesIPRanges(mock.Anything, cloudtypes.CloudProviderName(provider), []string{"service"}).
			Return(map[string][]string{"service": ranges}, nil).Once()
		responses, err := disruption.initialSafetyNets(context.Background())
		Expect(err).NotTo(HaveOccurred())
		if wantCaught {
			Expect(responses).To(ContainElement(ContainSubstring("no Host or Service filters")))
		} else {
			Expect(responses).To(BeEmpty())
		}
		Expect(disruption.Spec.Network.Hosts).To(BeNil())
	},
		Entry("AWS", "AWS", []string{"192.0.2.0/24"}, false),
		Entry("GCP", "GCP", []string{"192.0.2.0/24"}, false),
		Entry("Datadog", "Datadog", []string{"192.0.2.0/24"}, false),
		Entry("empty resolution still requires the unsafe override", "AWS", []string{}, true),
	)

	It("preserves the explicit network safety override without resolving cloud services", func() {
		services := []NetworkDisruptionCloudServiceSpec{{ServiceName: "service"}}
		cloudServicesProvidersManager = cloudservice.NewCloudServicesProvidersManagerMock(GinkgoT())
		disruption := &Disruption{Spec: DisruptionSpec{
			Level:      "pod",
			Network:    &NetworkDisruptionSpec{Drop: 100, Cloud: &NetworkDisruptionCloudSpec{AWSServiceList: &services}},
			Unsafemode: &UnsafemodeSpec{DisableCountTooLarge: true, DisableNeitherHostNorPort: true},
		}}
		responses, err := disruption.initialSafetyNets(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(responses).To(BeEmpty())
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cloudservice

import (
	"reflect"
	"testing"

	"github.com/DataDog/chaos-controller/cloudservice/gcp"
	"github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloudservice Manager Suite")
}

var _ = Describe("New function", func() {
	Context("Create New success", func() {
		logger, _ := log.NewZapLogger()

		manager, err := New(logger, types.CloudProviderConfigs{
			PullInterval: "1m",
		})

		It("should have parsed once", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that we have all cloud managed services")
			Expect(manager.cloudProviders[types.CloudProviderAWS]).ToNot(BeNil())
			Expect(manager.cloudProviders[types.CloudProviderGCP]).ToNot(BeNil())

			By("Ensuring that the ips are parsed")
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangeInfo.IPRanges).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangeInfo.IPRanges).ToNot(BeEmpty())

			By("Ensuring that we have a service list for every cloud provider")
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangeInfo.ServiceList).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangeInfo.ServiceList).ToNot(BeEmpty())
		})
	})

	Context("Pull new ip ranges from aws and gcp", func() {
		logger, _ := log.NewZapLogger()

		manager := &CloudServicesProvidersManager{
			log: logger,
			cloudProviders: map[types.CloudProviderName]*CloudServicesProvider{
				types.CloudProviderAWS: {
					CloudProviderIPRangeManager: NewCloudServiceMock(
						true,
						nil,
						"1",
						[]string{"S3", "EC2"},
						map[string][]string{
							"S3": {
								"1.2.3.0/24",
								"2.2.3.0/24",
							},
							"EC2": {
								"4.2.3.0/24",
								"5.2.3.0/24",
							},
						},
						nil,
					),
					Conf: types.CloudProviderConfig{
						IPRangesURL: "https://ip-ranges.amazonaws.com/ip-ranges.json",
					},
				},
				types.CloudProviderGCP: {
					CloudProviderIPRangeManager: NewCloudServiceMock(
						true,
						nil,
						"1",
						[]string{gcp.GoogleCloudService},
						map[string][]string{
							gcp.GoogleCloudService: {
								"6.2.3.0/24",
								"7.2.3.0/24",
								"8.2.3.0/24",
							},
						},
						nil,
					),
					Conf: types.CloudProviderConfig{
						IPRangesURL: "https://www.gstatic.com/ipranges/goog.json", // General IP Ranges from Google, contains some API ip ranges
					},
				},
			},
		}

		err := manager.PullIPRanges()
		It("should have parsed successfully the service list", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that we have a service list for every cloud provider")
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangeInfo.ServiceList).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangeInfo.ServiceList).ToNot(BeEmpty())

			By("Ensuring aws service list is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderAWS].IPRangeInfo.ServiceList, []string{
				"S3",
				"EC2",
			})).To(BeTrue())
			Expect(reflect.DeepEqual(manager.GetServiceList(types.CloudProviderAWS), []string{
				"S3",
				"EC2",
			})).To(BeTrue())

			By("Ensuring gcp service list is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderGCP].IPRangeInfo.ServiceList, []string{
				gcp.GoogleCloudService,
			})).To(BeTrue())
			Expect(reflect.DeepEqual(manager.GetServiceList(types.CloudProviderGCP), []string{
				gcp.GoogleCloudService,
			})).To(BeTrue())
		})

		It("should have parsed successfully the ip ranges map", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that we have an ip ranges map for every cloud provider")
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangeInfo.IPRanges).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangeInfo.IPRanges).ToNot(BeEmpty())

			By("Ensuring aws ip ranges map is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderAWS].IPRangeInfo.IPRanges, map[string][]string{
				"S3": {
					"1.2.3.0/24",
					"2.2.3.0/24",
				},
				"EC2": {
					"4.2.3.0/24",
					"5.2.3.0/24",
				},
			})).To(BeTrue())

			By("Ensuring it returns the right ip ranges map when using the GetServicesIPRanges function")
			ipRanges, err := manager.GetServicesIPRanges(types.CloudProviderAWS, []string{"S3", "EC2"})
			Expect(err).To(BeNil())
			Expect(reflect.DeepEqual(ipRanges, map[string][]string{
				"S3": {
					"1.2.3.0/24",
					"2.2.3.0/24",
				},
				"EC2": {
					"4.2.3.0/24",
					"5.2.3.0/24",
				},
			})).To(BeTrue())

			By("Ensuring gcp ip ranges map is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderGCP].IPRangeInfo.IPRanges, map[string][]string{
				gcp.GoogleCloudService: {
					"6.2.3.0/24",
					"7.2.3.0/24",
					"8.2.3.0/24",
				},
			})).To(BeTrue())

			By("Ensuring it returns the right ip ranges map when using the GetServicesIPRanges function")
			ipRanges, err = manager.GetServicesIPRanges(types.CloudProviderGCP, []string{gcp.GoogleCloudService})
			Expect(err).To(BeNil())
			Expect(reflect.DeepEqual(ipRanges, map[string][]string{
				gcp.GoogleCloudService: {
					"6.2.3.0/24",
					"7.2.3.0/24",
					"8.2.3.0/24",
				},
			})).To(BeTrue())
		})
	})
})

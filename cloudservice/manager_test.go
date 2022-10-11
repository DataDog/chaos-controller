package cloudservice

import (
	"reflect"
	"testing"

	"github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cloudservice Manager Suite")
}

var _ = BeforeSuite(func() {
})

var _ = AfterSuite(func() {
})

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
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangesInfo.IPRanges).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangesInfo.IPRanges).ToNot(BeEmpty())

			By("Ensuring that we have a service list for every cloud provider")
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangesInfo.ServiceList).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangesInfo.ServiceList).ToNot(BeEmpty())
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
						"1",
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
						"1",
						map[string][]string{
							"Google Cloud": {
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
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangesInfo.ServiceList).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangesInfo.ServiceList).ToNot(BeEmpty())

			By("Ensuring aws service list is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderAWS].IPRangesInfo.ServiceList, []string{
				"S3",
				"EC2",
			})).To(BeTrue())
			Expect(reflect.DeepEqual(manager.GetServiceList(types.CloudProviderAWS), []string{
				"S3",
				"EC2",
			})).To(BeTrue())

			By("Ensuring gcp service list is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderGCP].IPRangesInfo.ServiceList, []string{
				"Google Cloud",
			})).To(BeTrue())
			Expect(reflect.DeepEqual(manager.GetServiceList(types.CloudProviderGCP), []string{
				"Google Cloud",
			})).To(BeTrue())
		})

		It("should have parsed successfully the ip ranges map", func() {
			By("Ensuring that no error was thrown")
			Expect(err).To(BeNil())

			By("Ensuring that we have an ip ranges map for every cloud provider")
			Expect(manager.cloudProviders[types.CloudProviderAWS].IPRangesInfo.IPRanges).ToNot(BeEmpty())
			Expect(manager.cloudProviders[types.CloudProviderGCP].IPRangesInfo.IPRanges).ToNot(BeEmpty())

			By("Ensuring aws ip ranges map is populated with the right information")
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderAWS].IPRangesInfo.IPRanges, map[string][]string{
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
			Expect(reflect.DeepEqual(manager.cloudProviders[types.CloudProviderGCP].IPRangesInfo.IPRanges, map[string][]string{
				"Google Cloud": {
					"6.2.3.0/24",
					"7.2.3.0/24",
					"8.2.3.0/24",
				},
			})).To(BeTrue())

			By("Ensuring it returns the right ip ranges map when using the GetServicesIPRanges function")
			ipRanges, err = manager.GetServicesIPRanges(types.CloudProviderGCP, []string{"Google Cloud"})
			Expect(err).To(BeNil())
			Expect(reflect.DeepEqual(ipRanges, map[string][]string{
				"Google Cloud": {
					"6.2.3.0/24",
					"7.2.3.0/24",
					"8.2.3.0/24",
				},
			})).To(BeTrue())
		})
	})
})

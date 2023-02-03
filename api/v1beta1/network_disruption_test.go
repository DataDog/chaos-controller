package v1beta1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetworkDisruption Format test", func() {
	When("NetworkDisruption has filters", func() {
		It("expects good formatting for multiple hosts", func() {
			disruptionSpec := NetworkDisruptionSpec{
				Hosts: []NetworkDisruptionHostSpec{
					{
						Host: "1.2.3.4",
						Port: 9000,
					},
					{
						Host: "2.2.3.4",
						Port: 8000,
						Flow: "ingress",
					},
				},
				Drop: 100,
			}

			expected := "Network disruption dropping 100% of the traffic going to 1.2.3.4:9000 and coming from 2.2.3.4:8000"
			result := disruptionSpec.Format()

			Expect(result).To(Equal(expected))
		})

		It("expects good formatting for multiple services", func() {
			disruptionSpec := NetworkDisruptionSpec{
				Services: []NetworkDisruptionServiceSpec{
					{
						Name:      "demo-service",
						Namespace: "demo-namespace",
					},
					{
						Name:      "demo-worker",
						Namespace: "demo-namespace",
					},
				},
				Corrupt: 100,
			}

			expected := "Network disruption corrupting 100% of the traffic going to demo-service/demo-namespace and going to demo-worker/demo-namespace"
			result := disruptionSpec.Format()

			Expect(result).To(Equal(expected))
		})

		It("expects good formatting for cloud network disruption", func() {
			disruptionSpec := NetworkDisruptionSpec{
				Cloud: &NetworkDisruptionCloudSpec{
					AWSServiceList: &[]NetworkDisruptionCloudServiceSpec{
						{
							ServiceName: "S3",
						},
					},
					DatadogServiceList: &[]NetworkDisruptionCloudServiceSpec{
						{
							ServiceName: "synthetics",
						},
					},
				},
				Delay:       100,
				DelayJitter: 50,
			}

			expected := "Network disruption delaying of 100ms the traffic with 50ms of delay jitter going to S3 and going to synthetics"
			result := disruptionSpec.Format()

			Expect(result).To(Equal(expected))
		})

		It("expects good formatting for a big network disruption", func() {
			disruptionSpec := NetworkDisruptionSpec{
				Hosts: []NetworkDisruptionHostSpec{
					{
						Host: "1.2.3.4",
						Port: 9000,
					},
					{
						Host: "2.2.3.4",
						Port: 8000,
						Flow: "ingress",
					},
				},
				Services: []NetworkDisruptionServiceSpec{
					{
						Name:      "demo-service",
						Namespace: "demo-namespace",
					},
					{
						Name:      "demo-worker",
						Namespace: "demo-namespace",
					},
				},
				Cloud: &NetworkDisruptionCloudSpec{
					AWSServiceList: &[]NetworkDisruptionCloudServiceSpec{
						{
							ServiceName: "S3",
						},
					},
					DatadogServiceList: &[]NetworkDisruptionCloudServiceSpec{
						{
							ServiceName: "synthetics",
						},
					},
				},
				Corrupt: 100,
			}

			expected := "Network disruption corrupting 100% of the traffic going to 1.2.3.4:9000, coming from 2.2.3.4:8000, going to demo-service/demo-namespace, going to demo-worker/demo-namespace, going to S3 and going to synthetics"
			result := disruptionSpec.Format()

			Expect(result).To(Equal(expected))
		})

		It("expects no formatting for empty network disruption", func() {
			disruptionSpec := NetworkDisruptionSpec{
				Hosts:    []NetworkDisruptionHostSpec{},
				Services: []NetworkDisruptionServiceSpec{},
			}

			expected := ""
			result := disruptionSpec.Format()

			Expect(result).To(Equal(expected))
		})
	})
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cloudservice

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice/gcp"
	"github.com/DataDog/chaos-controller/cloudservice/types"
	"github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CloudService Manager Suite")
}

const (
	AWSURL     = "https://ip-ranges.amazonaws.com/ip-ranges.json"
	GCPURL     = "https://www.gstatic.com/ipranges/goog.json"
	DatadogURL = "https://ip-ranges.datadoghq.com/"
)

var _ = Describe("New function", func() {

	var (
		configs              types.CloudProviderConfigs
		manager              CloudServicesProvidersManager
		httpRoundTripperMock *mocks.RoundTripperMock
	)

	BeforeEach(func() {
		configs = types.CloudProviderConfigs{
			DisableAll:   false,
			PullInterval: time.Minute,
			AWS: types.CloudProviderConfig{
				Enabled:     true,
				IPRangesURL: AWSURL,
			},
			GCP: types.CloudProviderConfig{
				Enabled:     true,
				IPRangesURL: GCPURL,
			},
			Datadog: types.CloudProviderConfig{
				Enabled:     true,
				IPRangesURL: DatadogURL,
			},
		}
		httpRoundTripperMock = mocks.NewRoundTripperMock(GinkgoT())
		httpRoundTripperMock.EXPECT().RoundTrip(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			}, nil
		}).Maybe()
	})

	JustBeforeEach(func(ctx SpecContext) {
		var err error

		// Add logger to context for testing
		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		httpClient := http.Client{
			Transport: httpRoundTripperMock,
		}
		manager, err = New(ctxWithLogger, configs, &httpClient)

		By("Ensuring that no error was thrown")
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Creating a new manager with all providers enabled", func() {
		BeforeEach(func() {
			// Arrange
			httpRoundTripperMock = mocks.NewRoundTripperMock(GinkgoT())
			httpRoundTripperMock.EXPECT().RoundTrip(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
				var body []byte
				switch request.URL.String() {
				case AWSURL:
					body = []byte(`{
  "syncToken": "1693194189",
  "createDate": "2023-08-28-03-43-09",
  "prefixes": [
    {
      "ip_prefix": "3.2.34.0/26",
      "region": "af-south-1",
      "service": "ROUTE53_RESOLVER",
      "network_border_group": "af-south-1"
    }
  ]
}`)
				case GCPURL:
					body = []byte(`{
  "syncToken": "1693209970630",
  "creationTime": "2023-08-28T01:06:10.63098",
  "prefixes": [{
    "ipv4Prefix": "8.8.4.0/24"
  }]
}`)
				case DatadogURL:
					body = []byte(`{
    "version": 54,
    "modified": "2023-07-14-00-00-00",
    "agents": {
        "prefixes_ipv4": [
            "3.233.144.0/20"
        ],
        "prefixes_ipv6": [
            "2600:1f18:24e6:b900::/56"
        ]
    }
}`)
				default:
					return nil, errors.New("unknown URL")
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}).Maybe()
		})

		It("should have parsed once", func() {
			awsProvider := manager.GetProviderByName(types.CloudProviderAWS)
			GCPProvider := manager.GetProviderByName(types.CloudProviderGCP)
			DatadogProvider := manager.GetProviderByName(types.CloudProviderDatadog)

			By("Ensuring that we have all cloud managed services")
			Expect(awsProvider).ToNot(BeNil())
			Expect(GCPProvider).ToNot(BeNil())
			Expect(DatadogProvider).ToNot(BeNil())

			By("Ensuring that the ips are parsed")
			Expect(awsProvider.IPRangeInfo.IPRanges).ToNot(BeEmpty())
			Expect(GCPProvider.IPRangeInfo.IPRanges).ToNot(BeEmpty())
			Expect(DatadogProvider.IPRangeInfo.IPRanges).ToNot(BeEmpty())

			By("Ensuring that we have a service list for every cloud provider")
			Expect(awsProvider.IPRangeInfo.ServiceList).ToNot(BeEmpty())
			Expect(GCPProvider.IPRangeInfo.ServiceList).ToNot(BeEmpty())
			Expect(DatadogProvider.IPRangeInfo.ServiceList).ToNot(BeEmpty())
		})
	})

	Context("Creating a new manager with one provider disabled", func() {
		BeforeEach(func() {
			configs.AWS.Enabled = false
		})

		It("should have parsed once", func() {
			By("Ensuring that we have all cloud managed services")
			Expect(manager.GetProviderByName(types.CloudProviderAWS)).To(BeNil())
			Expect(manager.GetProviderByName(types.CloudProviderGCP)).ToNot(BeNil())
			Expect(manager.GetProviderByName(types.CloudProviderDatadog)).ToNot(BeNil())
		})
	})

	Context("Creating a new manager with all providers disabled", func() {
		BeforeEach(func() {
			configs.DisableAll = true
		})

		It("should have parsed once", func() {
			By("Ensuring that we have all cloud managed services")
			Expect(manager.GetProviderByName(types.CloudProviderAWS)).To(BeNil())
			Expect(manager.GetProviderByName(types.CloudProviderGCP)).To(BeNil())
			Expect(manager.GetProviderByName(types.CloudProviderDatadog)).To(BeNil())
		})
	})

	Context("Creating a new manager with extra IP ranges for all providers", func() {
		BeforeEach(func() {
			configs.AWS.ExtraIPRanges = []string{"S3;10.10.0.0/16"}
			configs.GCP.ExtraIPRanges = []string{"Google;10.30.0.0/16"}
			configs.Datadog.ExtraIPRanges = []string{"synthetics;10.20.0.0/16"}

			httpRoundTripperMock = mocks.NewRoundTripperMock(GinkgoT())
			httpRoundTripperMock.EXPECT().RoundTrip(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
				var body []byte
				switch request.URL.String() {
				case AWSURL:
					body = []byte(`{
  "syncToken": "1693194189",
  "createDate": "2023-08-28-03-43-09",
  "prefixes": [
    {
      "ip_prefix": "1.2.3.0/24",
      "region": "af-south-1",
      "service": "S3",
      "network_border_group": "af-south-1"
    }
  ]
}`)
				case GCPURL:
					body = []byte(`{
  "syncToken": "1693209970630",
  "creationTime": "2023-08-28T01:06:10.63098",
  "prefixes": [{
    "ipv4Prefix": "6.2.3.0/24"
  }]
}`)
				case DatadogURL:
					body = []byte(`{
  "version": 54,
  "modified": "2023-07-14-00-00-00",
  "synthetics": {
    "prefixes_ipv4": [
      "3.233.144.0/20"
    ]
  }
}`)
				default:
					return nil, errors.New("unknown URL")
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}).Maybe()
		})

		It("should expose configured extra IP ranges for AWS, GCP, and Datadog", func(ctx SpecContext) {
			awsRanges, err := manager.GetServicesIPRanges(ctx, types.CloudProviderAWS, []string{"S3"})
			Expect(err).ToNot(HaveOccurred())
			Expect(awsRanges["S3"]).To(Equal([]string{"1.2.3.0/24", "10.10.0.0/16"}))

			gcpRanges, err := manager.GetServicesIPRanges(ctx, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
			Expect(err).ToNot(HaveOccurred())
			Expect(gcpRanges[gcp.GoogleCloudService]).To(Equal([]string{"6.2.3.0/24", "10.30.0.0/16"}))

			datadogRanges, err := manager.GetServicesIPRanges(ctx, types.CloudProviderDatadog, []string{"synthetics"})
			Expect(err).ToNot(HaveOccurred())
			Expect(datadogRanges["synthetics"]).To(Equal([]string{"3.233.144.0/20", "10.20.0.0/16"}))
		})
	})

	Context("Pull new ip ranges from aws and gcp", func() {
		BeforeEach(func() {
			// Arrange
			httpRoundTripperMock = mocks.NewRoundTripperMock(GinkgoT())
			httpRoundTripperMock.EXPECT().RoundTrip(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
				var body []byte
				switch request.URL.String() {
				case AWSURL:
					body = []byte(`{
  "syncToken": "1693194189",
  "createDate": "2023-08-28-03-43-09",
  "prefixes": [
    {
      "ip_prefix": "1.2.3.0/24",
      "region": "af-south-1",
      "service": "S3",
      "network_border_group": "af-south-1"
    },
    {
      "ip_prefix": "2.2.3.0/24",
      "region": "af-south-1",
      "service": "S3",
      "network_border_group": "af-south-1"
    },
    {
      "ip_prefix": "4.2.3.0/24",
      "region": "af-south-1",
      "service": "EC2",
      "network_border_group": "af-south-1"
    },
    {
      "ip_prefix": "5.2.3.0/24",
      "region": "af-south-1",
      "service": "EC2",
      "network_border_group": "af-south-1"
    }
  ]
}`)
				case GCPURL:
					body = []byte(`{
  "syncToken": "1693209970630",
  "creationTime": "2023-08-28T01:06:10.63098",
  "prefixes": [{
    "ipv4Prefix": "6.2.3.0/24"
  },
  {
	"ipv4Prefix": "7.2.3.0/24"
  },
  {
	"ipv4Prefix": "8.2.3.0/24"
  }]
}`)
				case DatadogURL:
					body = []byte(`{
    "version": 54,
    "modified": "2023-07-14-00-00-00",
    "agents": {
        "prefixes_ipv4": [
            "3.233.144.0/20"
        ],
        "prefixes_ipv6": [
            "2600:1f18:24e6:b900::/56"
        ]
    }
}`)
				default:
					return nil, errors.New("unknown URL")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(body)),
				}, nil
			}).Maybe()

			configs = types.CloudProviderConfigs{
				DisableAll:   false,
				PullInterval: time.Minute,
				AWS: types.CloudProviderConfig{
					Enabled:     true,
					IPRangesURL: "https://ip-ranges.amazonaws.com/ip-ranges.json",
				},
				GCP: types.CloudProviderConfig{
					Enabled:     true,
					IPRangesURL: "https://www.gstatic.com/ipranges/goog.json",
				},
			}

		})

		JustBeforeEach(func(ctx SpecContext) {
			// Action
			err := manager.PullIPRanges(ctx)

			By("Ensuring that no error was thrown")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should have parsed successfully the service list", func(ctx SpecContext) {
			awsProvider := manager.GetProviderByName(types.CloudProviderAWS)
			GCPProvider := manager.GetProviderByName(types.CloudProviderGCP)

			By("Ensuring that we have a service list for every cloud provider")
			Expect(awsProvider.IPRangeInfo.ServiceList).ToNot(BeEmpty())
			Expect(GCPProvider.IPRangeInfo.ServiceList).ToNot(BeEmpty())

			By("Ensuring aws service list is populated with the right information")
			Expect(reflect.DeepEqual(awsProvider.IPRangeInfo.ServiceList, []string{
				"S3",
				"EC2",
			})).To(BeTrue())
			Expect(reflect.DeepEqual(manager.GetServiceList(ctx, types.CloudProviderAWS), []string{
				"S3",
				"EC2",
			})).To(BeTrue())

			By("Ensuring gcp service list is populated with the right information")
			Expect(reflect.DeepEqual(GCPProvider.IPRangeInfo.ServiceList, []string{
				gcp.GoogleCloudService,
			})).To(BeTrue())
			Expect(reflect.DeepEqual(manager.GetServiceList(ctx, types.CloudProviderGCP), []string{
				gcp.GoogleCloudService,
			})).To(BeTrue())
		})

		It("should have parsed successfully the ip ranges map", func(ctx SpecContext) {
			awsProvider := manager.GetProviderByName(types.CloudProviderAWS)
			GCPProvider := manager.GetProviderByName(types.CloudProviderGCP)

			By("Ensuring that we have an ip ranges map for every cloud provider")
			Expect(awsProvider.IPRangeInfo.IPRanges).ToNot(BeEmpty())
			Expect(GCPProvider.IPRangeInfo.IPRanges).ToNot(BeEmpty())

			By("Ensuring aws ip ranges map is populated with the right information")
			Expect(reflect.DeepEqual(awsProvider.IPRangeInfo.IPRanges, map[string][]string{
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
			ipRanges, err := manager.GetServicesIPRanges(ctx, types.CloudProviderAWS, []string{"S3", "EC2"})
			Expect(err).ToNot(HaveOccurred())
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
			Expect(reflect.DeepEqual(GCPProvider.IPRangeInfo.IPRanges, map[string][]string{
				gcp.GoogleCloudService: {
					"6.2.3.0/24",
					"7.2.3.0/24",
					"8.2.3.0/24",
				},
			})).To(BeTrue())

			By("Ensuring it returns the right ip ranges map when using the GetServicesIPRanges function")
			ipRanges, err = manager.GetServicesIPRanges(ctx, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
			Expect(err).ToNot(HaveOccurred())
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

var _ = Describe("Pulling IP ranges on parse error", func() {
	It("returns an error without panicking when New receives an unparseable 200 body with GCP extra IP ranges", func(ctx SpecContext) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `garbage{`)
		}))
		DeferCleanup(server.Close)

		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		var (
			manager CloudServicesProvidersManager
			newErr  error
		)
		Expect(func() {
			manager, newErr = New(ctxWithLogger, types.CloudProviderConfigs{
				PullInterval: time.Minute,
				GCP: types.CloudProviderConfig{
					Enabled:       true,
					IPRangesURL:   server.URL,
					ExtraIPRanges: []string{"Google;10.9.0.0/16"},
				},
			}, server.Client())
		}).ToNot(Panic())

		Expect(newErr).To(HaveOccurred())
		Expect(manager).To(BeNil())
	})

	It("retains previously pulled ranges including extras when a refresh returns an unparseable 200 body", func(ctx SpecContext) {
		var corrupt atomic.Bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if corrupt.Load() {
				_, _ = fmt.Fprint(w, `garbage{`)
				return
			}
			_, _ = fmt.Fprint(w, `{"syncToken":"1","prefixes":[{"ipv4Prefix":"192.0.2.0/24"}]}`)
		}))
		DeferCleanup(server.Close)

		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		manager, err := New(ctxWithLogger, types.CloudProviderConfigs{
			PullInterval: time.Minute,
			GCP: types.CloudProviderConfig{
				Enabled:       true,
				IPRangesURL:   server.URL,
				ExtraIPRanges: []string{"Google;10.9.0.0/16"},
			},
		}, server.Client())
		Expect(err).ToNot(HaveOccurred())

		corrupt.Store(true)
		refreshErr := manager.PullIPRanges(ctxWithLogger)
		Expect(refreshErr).To(HaveOccurred())

		ranges, err := manager.GetServicesIPRanges(ctxWithLogger, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
		Expect(err).ToNot(HaveOccurred())
		Expect(ranges[gcp.GoogleCloudService]).To(Equal([]string{"192.0.2.0/24", "10.9.0.0/16"}))
	})

	It("retains previously pulled ranges including extras when ConvertToGenericIPRanges fails on refresh", func(ctx SpecContext) {
		var corrupt atomic.Bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if corrupt.Load() {
				_, _ = fmt.Fprint(w, `garbage{`)
				return
			}
			_, _ = fmt.Fprint(w, `{"syncToken":"1","prefixes":[{"ipv4Prefix":"192.0.2.0/24"}]}`)
		}))
		DeferCleanup(server.Close)

		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		manager, err := New(ctxWithLogger, types.CloudProviderConfigs{
			PullInterval: time.Minute,
			GCP: types.CloudProviderConfig{
				Enabled:       true,
				IPRangesURL:   server.URL,
				ExtraIPRanges: []string{"Google;10.9.0.0/16"},
			},
		}, server.Client())
		Expect(err).ToNot(HaveOccurred())

		converterMock := NewCloudProviderIPRangeManagerMock(GinkgoT())
		converterMock.EXPECT().IsNewVersion([]byte(`garbage{`), "1").Return(true, nil)
		converterMock.EXPECT().ConvertToGenericIPRanges([]byte(`garbage{`)).Return(nil, errors.New("parse error"))
		manager.GetProviderByName(types.CloudProviderGCP).CloudProviderIPRangeManager = converterMock

		corrupt.Store(true)
		var refreshErr error
		Expect(func() {
			refreshErr = manager.PullIPRanges(ctxWithLogger)
		}).ToNot(Panic())
		Expect(refreshErr).To(HaveOccurred())
		Expect(refreshErr.Error()).To(ContainSubstring("parse error"))

		ranges, err := manager.GetServicesIPRanges(ctxWithLogger, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
		Expect(err).ToNot(HaveOccurred())
		Expect(ranges[gcp.GoogleCloudService]).To(Equal([]string{"192.0.2.0/24", "10.9.0.0/16"}))
	})

	It("retains previously pulled ranges without extras when ConvertToGenericIPRanges fails on refresh", func(ctx SpecContext) {
		var corrupt atomic.Bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if corrupt.Load() {
				_, _ = fmt.Fprint(w, `garbage{`)
				return
			}
			_, _ = fmt.Fprint(w, `{"syncToken":"1","prefixes":[{"ipv4Prefix":"192.0.2.0/24"}]}`)
		}))
		DeferCleanup(server.Close)

		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		manager, err := New(ctxWithLogger, types.CloudProviderConfigs{
			PullInterval: time.Minute,
			GCP: types.CloudProviderConfig{
				Enabled:     true,
				IPRangesURL: server.URL,
			},
		}, server.Client())
		Expect(err).ToNot(HaveOccurred())

		converterMock := NewCloudProviderIPRangeManagerMock(GinkgoT())
		converterMock.EXPECT().IsNewVersion([]byte(`garbage{`), "1").Return(true, nil)
		converterMock.EXPECT().ConvertToGenericIPRanges([]byte(`garbage{`)).Return(nil, errors.New("parse error"))
		manager.GetProviderByName(types.CloudProviderGCP).CloudProviderIPRangeManager = converterMock

		corrupt.Store(true)
		refreshErr := manager.PullIPRanges(ctxWithLogger)
		Expect(refreshErr).To(HaveOccurred())
		Expect(refreshErr.Error()).To(ContainSubstring("parse error"))

		var (
			ranges map[string][]string
			getErr error
		)
		Expect(func() {
			ranges, getErr = manager.GetServicesIPRanges(ctxWithLogger, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
		}).ToNot(Panic())
		Expect(getErr).ToNot(HaveOccurred())
		Expect(ranges[gcp.GoogleCloudService]).To(Equal([]string{"192.0.2.0/24"}))
	})

	It("handles a converter returning a non-nil info with a nil IPRanges map", func(ctx SpecContext) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"syncToken":"1","prefixes":[{"ipv4Prefix":"192.0.2.0/24"}]}`)
		}))
		DeferCleanup(server.Close)

		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		manager, err := New(ctxWithLogger, types.CloudProviderConfigs{
			PullInterval: time.Minute,
			GCP: types.CloudProviderConfig{
				Enabled:       true,
				IPRangesURL:   server.URL,
				ExtraIPRanges: []string{"Google;10.9.0.0/16"},
			},
		}, server.Client())
		Expect(err).ToNot(HaveOccurred())

		converterMock := NewCloudProviderIPRangeManagerMock(GinkgoT())
		converterMock.EXPECT().IsNewVersion(mock.Anything, "1").Return(true, nil)
		converterMock.EXPECT().ConvertToGenericIPRanges(mock.Anything).Return(&types.CloudProviderIPRangeInfo{
			Version:     "2",
			ServiceList: []string{gcp.GoogleCloudService},
			IPRanges:    nil,
		}, nil)
		manager.GetProviderByName(types.CloudProviderGCP).CloudProviderIPRangeManager = converterMock

		var refreshErr error
		Expect(func() {
			refreshErr = manager.PullIPRanges(ctxWithLogger)
		}).ToNot(Panic())
		Expect(refreshErr).ToNot(HaveOccurred())

		ranges, err := manager.GetServicesIPRanges(ctxWithLogger, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
		Expect(err).ToNot(HaveOccurred())
		Expect(ranges[gcp.GoogleCloudService]).To(Equal([]string{"10.9.0.0/16"}))
	})

	It("retains previously pulled ranges without partial updates when an extra IP range entry is invalid", func(ctx SpecContext) {
		var versionTwo atomic.Bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if versionTwo.Load() {
				_, _ = fmt.Fprint(w, `{"syncToken":"2","prefixes":[{"ipv4Prefix":"198.51.100.0/24"}]}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"syncToken":"1","prefixes":[{"ipv4Prefix":"192.0.2.0/24"}]}`)
		}))
		DeferCleanup(server.Close)

		logger, err := log.NewZapLogger()
		Expect(err).ToNot(HaveOccurred())
		ctxWithLogger := log.WithLogger(ctx, logger)

		manager, err := New(ctxWithLogger, types.CloudProviderConfigs{
			PullInterval: time.Minute,
			GCP: types.CloudProviderConfig{
				Enabled:       true,
				IPRangesURL:   server.URL,
				ExtraIPRanges: []string{"Google;10.9.0.0/16"},
			},
		}, server.Client())
		Expect(err).ToNot(HaveOccurred())

		manager.GetProviderByName(types.CloudProviderGCP).Conf.ExtraIPRanges = []string{"Google;10.10.0.0/16", "invalid-entry"}
		versionTwo.Store(true)

		refreshErr := manager.PullIPRanges(ctxWithLogger)
		Expect(refreshErr).To(HaveOccurred())
		Expect(refreshErr.Error()).To(ContainSubstring(`invalid extra IP range entry "invalid-entry": expected service;iprange[;iprange...]`))

		ranges, err := manager.GetServicesIPRanges(ctxWithLogger, types.CloudProviderGCP, []string{gcp.GoogleCloudService})
		Expect(err).ToNot(HaveOccurred())
		Expect(ranges[gcp.GoogleCloudService]).To(Equal([]string{"192.0.2.0/24", "10.9.0.0/16"}))
	})
})

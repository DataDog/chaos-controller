// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package cloudservice

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice/aws"
	"github.com/DataDog/chaos-controller/cloudservice/datadog"
	"github.com/DataDog/chaos-controller/cloudservice/gcp"
	"github.com/DataDog/chaos-controller/cloudservice/types"

	"go.uber.org/zap"
)

// CloudServicesProvidersManager represents an interface for managing cloud service providers and their IP ranges.
type CloudServicesProvidersManager interface {
	// GetServiceList returns a list of service names provided by the specified cloud provider.
	GetServiceList(cloudProviderName types.CloudProviderName) []string

	// GetServicesIPRanges retrieves IP ranges for the specified services provided by the given cloud provider.
	GetServicesIPRanges(cloudProviderName types.CloudProviderName, serviceNames []string) (map[string][]string, error)

	// PullIPRanges triggers the manual pulling of IP ranges for all cloud providers.
	PullIPRanges() error

	// StartPeriodicPull starts the periodic process of pulling IP ranges from cloud providers.
	StartPeriodicPull()

	// StopPeriodicPull stops the periodic process of pulling IP ranges from cloud providers.
	StopPeriodicPull()

	// GetProviderByName retrieves the cloud services provider instance by its name.
	GetProviderByName(name types.CloudProviderName) *CloudServicesProvider
}

type cloudServicesProvidersManager struct {
	cloudProviders       map[types.CloudProviderName]*CloudServicesProvider
	log                  *zap.SugaredLogger
	stopPeriodicPull     chan bool
	periodicPullInterval time.Duration
	client               *http.Client
}

// CloudServicesProvider Data and ip ranges manager of one cloud provider
type CloudServicesProvider struct {
	// CloudProviderIPRangeManager is responsible for managing IP ranges for the cloud provider.
	CloudProviderIPRangeManager CloudProviderIPRangeManager

	// IPRangeInfo stores information about the IP ranges of the cloud services provided by the cloud provider.
	IPRangeInfo *types.CloudProviderIPRangeInfo

	// Conf contains the configuration settings for the cloud services provider.
	Conf types.CloudProviderConfig
}

// CloudProviderIPRangeManager Methods to verify and transform a specifid ip ranges list from a provider
type CloudProviderIPRangeManager interface {
	// IsNewVersion checks whether a given IP range data in the form of bytes is a new version compared to a given version string.
	// It returns true if the data is a new version, otherwise false. An error is returned in case of any issues.
	IsNewVersion(ipRangeData []byte, version string) (bool, error)

	// ConvertToGenericIPRanges converts the given IP range data in the form of bytes to a generic CloudProviderIPRangeInfo structure.
	// It returns the converted IP range information or an error in case of any issues during conversion.
	ConvertToGenericIPRanges(ipRangeData []byte) (*types.CloudProviderIPRangeInfo, error)
}

// New creates a new instance of CloudServicesProvidersManager.
// It initializes the manager with cloud providers based on the configuration and sets up their IP range managers.
func New(log *zap.SugaredLogger, config types.CloudProviderConfigs, httpClientMock *http.Client) (CloudServicesProvidersManager, error) {
	manager := &cloudServicesProvidersManager{
		cloudProviders:       map[types.CloudProviderName]*CloudServicesProvider{},
		log:                  log,
		periodicPullInterval: config.PullInterval,
	}

	if httpClientMock == nil {
		manager.client = &http.Client{
			Timeout: time.Second * 10,
		}
	} else {
		manager.client = httpClientMock
	}

	// return an empty manager if all providers are disabled
	if config.DisableAll {
		log.Info("all cloud providers are disabled")

		return manager, nil
	}

	for _, cp := range types.AllCloudProviders {
		provider := &CloudServicesProvider{}

		switch cp {
		case types.CloudProviderAWS:
			provider.CloudProviderIPRangeManager = aws.New()
			provider.Conf.Enabled = config.AWS.Enabled
			provider.Conf.IPRangesURL = config.AWS.IPRangesURL
		case types.CloudProviderGCP:
			provider.CloudProviderIPRangeManager = gcp.New()
			provider.Conf.Enabled = config.GCP.Enabled
			provider.Conf.IPRangesURL = config.GCP.IPRangesURL
		case types.CloudProviderDatadog:
			provider.CloudProviderIPRangeManager = datadog.New()
			provider.Conf.Enabled = config.Datadog.Enabled
			provider.Conf.IPRangesURL = config.Datadog.IPRangesURL
		}

		if !provider.Conf.Enabled {
			log.Debugw("a cloud provider was disabled", "provider", cp)

			continue
		}

		manager.cloudProviders[cp] = provider
	}

	if err := manager.PullIPRanges(); err != nil {
		manager.log.Error(err)
		return nil, err
	}

	return manager, nil
}

// StartPeriodicPull go routine pulling every interval all ip ranges of all cloud providers set up.
func (s *cloudServicesProvidersManager) StartPeriodicPull() {
	s.log.Infow("starting periodic pull and parsing of the cloud provider ip ranges", "interval", s.periodicPullInterval.String())

	go func() {
		for {
			select {
			case closed := <-s.stopPeriodicPull:
				if closed {
					return
				}
			case <-time.After(s.periodicPullInterval):
				if err := s.PullIPRanges(); err != nil {
					s.log.Errorw("an error occurred when pulling IP ranges", "error", err)
				}
			}
		}
	}()
}

// StopPeriodicPull stop the goroutine pulling all ip ranges of all cloud providers
func (s *cloudServicesProvidersManager) StopPeriodicPull() {
	s.log.Infow("closing periodic pull and parsing of the cloud provider ip ranges")

	s.stopPeriodicPull <- true
}

// PullIPRanges pull all ip ranges of all cloud providers
func (s *cloudServicesProvidersManager) PullIPRanges() error {
	errorMessage := ""

	s.log.Infow("pull and parse of the cloud provider ip ranges")

	for cloudProviderName := range s.cloudProviders {
		if err := s.pullIPRangesPerCloudProvider(cloudProviderName); err != nil {
			errorMessage += fmt.Sprintf("could not get the new ip ranges from provider %s: %s\n", cloudProviderName, err.Error())
		}
	}

	s.log.Infow("finished pull and parse of the cloud provider ip ranges")

	if errorMessage != "" {
		return fmt.Errorf(errorMessage)
	}

	return nil
}

// GetServicesIPRanges with a given list of service names and cloud provider name, returns the list of ip ranges of those services
func (s *cloudServicesProvidersManager) GetServicesIPRanges(cloudProviderName types.CloudProviderName, serviceNames []string) (map[string][]string, error) {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil, fmt.Errorf("cloud provider %s is not configured or does not exist", cloudProviderName)
	}

	if len(serviceNames) == 0 {
		return nil, fmt.Errorf("no cloud service list provided for cloud provider %s", cloudProviderName)
	}

	IPRangeInfo := s.cloudProviders[cloudProviderName].IPRangeInfo
	IPRanges := map[string][]string{}

	for _, serviceName := range serviceNames {
		// if user has imputed the same service twice, we verify
		if _, ok := IPRanges[serviceName]; ok {
			continue
		}

		if _, ok := IPRangeInfo.IPRanges[serviceName]; !ok {
			return nil, fmt.Errorf("service %s from %s does not exist, available services are: %s", serviceName, cloudProviderName, strings.Join(s.GetServiceList(cloudProviderName), ", "))
		}

		IPRanges[serviceName] = IPRangeInfo.IPRanges[serviceName]
	}

	return IPRanges, nil
}

// GetServiceList return the list of services of a specific cloud provider. Mostly used in disruption creation validation
func (s *cloudServicesProvidersManager) GetServiceList(cloudProviderName types.CloudProviderName) []string {
	if s.cloudProviders[cloudProviderName] == nil || s.cloudProviders[cloudProviderName].IPRangeInfo == nil {
		return nil
	}

	return s.cloudProviders[cloudProviderName].IPRangeInfo.ServiceList
}

// GetProviderByName retrieves a CloudServicesProvider instance by its name from the manager's collection of cloud providers.
func (s *cloudServicesProvidersManager) GetProviderByName(name types.CloudProviderName) *CloudServicesProvider {
	return s.cloudProviders[name]
}

// pullIPRangesPerCloudProvider pull ip ranges of one cloud provider
func (s *cloudServicesProvidersManager) pullIPRangesPerCloudProvider(cloudProviderName types.CloudProviderName) error {
	provider := s.cloudProviders[cloudProviderName]
	if provider == nil {
		return fmt.Errorf("cloud provider %s does not exist", cloudProviderName)
	}

	s.log.Debugw("pulling ip ranges from provider", "provider", cloudProviderName)

	unparsedIPRange, err := s.requestIPRangesFromProvider(provider.Conf.IPRangesURL)
	if err != nil {
		return err
	}

	if provider.IPRangeInfo != nil {
		isNewVersion, err := provider.CloudProviderIPRangeManager.IsNewVersion(unparsedIPRange, provider.IPRangeInfo.Version)
		if err != nil {
			return err
		}

		if !isNewVersion {
			s.log.Debugw("no changes of ip ranges", "provider", cloudProviderName)
			s.log.Debugw("finished pulling new version", "provider", cloudProviderName)

			return nil
		}
	}

	provider.IPRangeInfo, err = provider.CloudProviderIPRangeManager.ConvertToGenericIPRanges(unparsedIPRange)

	return err
}

// requestIPRangesFromProvider launches a HTTP GET request to pull the ip range json file from a url
func (s *cloudServicesProvidersManager) requestIPRangesFromProvider(url string) ([]byte, error) {
	response, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err = response.Body.Close(); err != nil {
		return nil, err
	}

	return body, nil
}

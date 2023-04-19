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

// CloudServicesProvidersManager Manager used to pull and parse any provider ip ranges per service
type CloudServicesProvidersManager struct {
	cloudProviders       map[types.CloudProviderName]*CloudServicesProvider
	log                  *zap.SugaredLogger
	stopPeriodicPull     chan bool
	periodicPullInterval time.Duration
}

// CloudServicesProvider Data and ip ranges manager of one cloud provider
type CloudServicesProvider struct {
	CloudProviderIPRangeManager CloudProviderIPRangeManager
	IPRangeInfo                 *types.CloudProviderIPRangeInfo
	Conf                        types.CloudProviderConfig
}

// CloudProviderIPRangeManager Methods to verify and transform a specifid ip ranges list from a provider
type CloudProviderIPRangeManager interface {
	IsNewVersion([]byte, string) (bool, error)
	ConvertToGenericIPRanges([]byte) (*types.CloudProviderIPRangeInfo, error)
}

func New(log *zap.SugaredLogger, config types.CloudProviderConfigs) (*CloudServicesProvidersManager, error) {
	manager := &CloudServicesProvidersManager{
		cloudProviders:       map[types.CloudProviderName]*CloudServicesProvider{},
		log:                  log,
		periodicPullInterval: config.PullInterval,
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
func (s *CloudServicesProvidersManager) StartPeriodicPull() {
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
					s.log.Errorf(err.Error())
				}
			}
		}
	}()
}

// StopPeriodicPull stop the goroutine pulling all ip ranges of all cloud providers
func (s *CloudServicesProvidersManager) StopPeriodicPull() {
	s.log.Infow("closing periodic pull and parsing of the cloud provider ip ranges")

	s.stopPeriodicPull <- true
}

// PullIPRanges pull all ip ranges of all cloud providers
func (s *CloudServicesProvidersManager) PullIPRanges() error {
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
func (s *CloudServicesProvidersManager) GetServicesIPRanges(cloudProviderName types.CloudProviderName, serviceNames []string) (map[string][]string, error) {
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
func (s *CloudServicesProvidersManager) GetServiceList(cloudProviderName types.CloudProviderName) []string {
	if s.cloudProviders[cloudProviderName] == nil || s.cloudProviders[cloudProviderName].IPRangeInfo == nil {
		return nil
	}

	return s.cloudProviders[cloudProviderName].IPRangeInfo.ServiceList
}

// pullIPRangesPerCloudProvider pull ip ranges of one cloud provider
func (s *CloudServicesProvidersManager) pullIPRangesPerCloudProvider(cloudProviderName types.CloudProviderName) error {
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
func (s *CloudServicesProvidersManager) requestIPRangesFromProvider(url string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
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

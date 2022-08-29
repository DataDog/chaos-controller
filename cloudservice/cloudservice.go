// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022 Datadog, Inc.

package cloudservice

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice/aws"
	"github.com/DataDog/chaos-controller/cloudservice/types"

	"go.uber.org/zap"
)

// CloudProviderManager manager used to pull and parse any provider ip ranges per service
type CloudProviderManager struct {
	cloudProviders map[types.CloudProviderName]*CloudProviderData
	log            *zap.SugaredLogger
	close          chan bool
}

type CloudProviderData struct {
	CloudProvider CloudProvider
	IPRangeInfo   *types.CloudProviderIPRangeInfo
	ServiceList   []string
	Conf          types.CloudProviderConfig
}

// CloudProvider Describe CloudProvider's methods
type CloudProvider interface {
	// Check if the ip ranges pulled are newer than the one we already have
	IsNewVersion([]byte, types.CloudProviderIPRangeInfo) bool
	// From an unmarshalled json result of a provider to a generic ip range struct
	ConvertToGenericIPRanges([]byte) (*types.CloudProviderIPRangeInfo, error)
}

func New(log *zap.SugaredLogger, config types.CloudProviderConfigs) (*CloudProviderManager, error) {
	cloudProviderMap := map[types.CloudProviderName]*CloudProviderData{
		types.CloudProviderAWS: {
			CloudProvider: aws.New(),
			Conf:          config.Aws,
			IPRangeInfo:   nil,
			ServiceList:   nil,
		},
	}

	manager := &CloudProviderManager{
		cloudProviders: cloudProviderMap,
		log:            log,
	}

	if err := manager.PullIPRanges(); err != nil {
		return nil, err
	}

	return manager, nil
}

// StartPeriodicPull go routine pulling every interval all ip ranges of all cloud providers set up.
func (s *CloudProviderManager) StartPeriodicPull(interval time.Duration) {
	s.log.Debugw("starting periodic pull and parsing of the cloud provider ip ranges")

	go func() {
		for {
			select {
			case closed := <-s.close:
				if closed {
					return
				}
			case <-time.After(interval):
				if err := s.PullIPRanges(); err != nil {
					s.log.Errorf(err.Error())
				}
			}
		}
	}()
}

// StopPeriodicPull stop the goroutine pulling all ip ranges of all cloud providers
func (s *CloudProviderManager) StopPeriodicPull() {
	s.log.Debugw("closing periodic pull and parsing of the cloud provider ip ranges")

	s.close <- true
}

// PullIPRanges pull all ip ranges of all cloud providers
func (s *CloudProviderManager) PullIPRanges() error {
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

// GetServiceIPRanges with a given service name and cloud provider name, returns the list of ip ranges of this service
func (s *CloudProviderManager) GetServiceIPRanges(cloudProviderName types.CloudProviderName, serviceName string) ([]string, error) {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil, fmt.Errorf("this cloud provider does not exist")
	}

	ipRangeInfo := s.cloudProviders[cloudProviderName].IPRangeInfo

	if serviceName != "" {
		return ipRangeInfo.IPRanges[serviceName], nil
	}

	// if no service name is provided, we return all ip ranges of all services
	allIPRanges := []string{}
	for _, ipRanges := range ipRangeInfo.IPRanges {
		allIPRanges = append(allIPRanges, ipRanges...)
	}

	return allIPRanges, nil
}

// GetServiceList return the list of services of a specific cloud provider. Mostly used in disruption creation validation
func (s *CloudProviderManager) GetServiceList(cloudProviderName types.CloudProviderName) []string {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil
	}

	return s.cloudProviders[cloudProviderName].ServiceList
}

// ServiceExists verify if a service exists for a cloud provider
func (s *CloudProviderManager) ServiceExists(cloudProviderName types.CloudProviderName, serviceName string) bool {
	if s.cloudProviders[cloudProviderName] == nil {
		return false
	}

	ipRanges := s.cloudProviders[cloudProviderName].IPRangeInfo
	if ipRanges == nil {
		return false
	}

	if serviceName != "" {
		_, ok := ipRanges.IPRanges[serviceName]

		return ok
	}

	return true
}

// pullIPRangesPerCloudProvider pull all ip ranges of all cloud providers
func (s *CloudProviderManager) pullIPRangesPerCloudProvider(cloudProviderName types.CloudProviderName) error {
	provider := s.cloudProviders[cloudProviderName]
	if provider == nil {
		return fmt.Errorf("cloud provider %s does not exist", cloudProviderName)
	}

	s.log.Debugw("pulling ip ranges from provider", "provider", cloudProviderName)

	unparsedIPRange, err := s.requestIPRangesFromProvider(provider.Conf.IPRangesURL)
	if err != nil {
		return err
	}

	if provider.IPRangeInfo != nil && !provider.CloudProvider.IsNewVersion(unparsedIPRange, *provider.IPRangeInfo) {
		s.log.Debugw("no changes of ip ranges", "provider", cloudProviderName)
		s.log.Debugw("finished pulling new version", "provider", cloudProviderName)

		return nil
	}

	newIPRangeInfo, err := provider.CloudProvider.ConvertToGenericIPRanges(unparsedIPRange)
	if err != nil {
		return err
	}

	// We compute this into a list to indicate to the user which services are available on error during disruption creation
	provider.IPRangeInfo = newIPRangeInfo
	for service := range newIPRangeInfo.IPRanges {
		provider.ServiceList = append(provider.ServiceList, service)
	}

	return nil
}

// requestIPRangesFromProvider launches a HTTP GET request to pull the ip range json file from a url
func (s *CloudProviderManager) requestIPRangesFromProvider(url string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(response.Body)
}

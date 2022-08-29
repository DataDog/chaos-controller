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
	IpRangeInfo   *types.CloudProviderIpRangeInfo
	ServiceList   []string
	Conf          types.CloudProviderConfig
}

// CloudProvider Describe CloudProvider's methods
type CloudProvider interface {
	// Check if the ip ranges pulled are newer than the one we already have
	IsNewVersion([]byte, types.CloudProviderIpRangeInfo) bool
	// From an unmarshalled json result of a provider to a generic ip range struct
	ConvertToGenericIpRanges([]byte) (*types.CloudProviderIpRangeInfo, error)
}

func New(log *zap.SugaredLogger, config types.CloudProviderConfigs) (*CloudProviderManager, error) {
	cloudProviderMap := map[types.CloudProviderName]*CloudProviderData{
		types.CloudProviderAWS: {
			CloudProvider: aws.New(),
			Conf:          config.Aws,
			IpRangeInfo:   nil,
			ServiceList:   nil,
		},
	}

	manager := &CloudProviderManager{
		cloudProviders: cloudProviderMap,
		log:            log,
	}

	if err := manager.PullIpRanges(); err != nil {
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
				if err := s.PullIpRanges(); err != nil {
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

// PullIpRanges pull all ip ranges of all cloud providers
func (s *CloudProviderManager) PullIpRanges() error {
	errorMessage := ""

	s.log.Infow("pull and parse of the cloud provider ip ranges")

	for cloudProviderName := range s.cloudProviders {
		if err := s.pullIpRangesPerCloudProvider(cloudProviderName); err != nil {
			s.log.Errorf("ERROR")
			errorMessage += fmt.Sprintf("could not get the new ip ranges from provider %s: %s\n", cloudProviderName, err.Error())
		}
	}

	s.log.Infow("finished pull and parse of the cloud provider ip ranges")

	if errorMessage != "" {
		return fmt.Errorf(errorMessage)
	}

	return nil
}

// GetServiceIpRanges with a given service name and cloud provider name, returns the list of ip ranges of this service
func (s *CloudProviderManager) GetServiceIpRanges(cloudProviderName types.CloudProviderName, serviceName string) ([]string, error) {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil, fmt.Errorf("this cloud provider does not exist")
	}

	ipRangeInfo := s.cloudProviders[cloudProviderName].IpRangeInfo

	if serviceName != "" {
		return ipRangeInfo.IpRanges[serviceName], nil
	}

	// if no service name is provided, we return all ip ranges of all services
	allIpRanges := []string{}
	for _, ipRanges := range ipRangeInfo.IpRanges {
		allIpRanges = append(allIpRanges, ipRanges...)
	}

	return allIpRanges, nil
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

	ipRanges := s.cloudProviders[cloudProviderName].IpRangeInfo
	if ipRanges == nil {
		return false
	}

	if serviceName != "" {
		_, ok := ipRanges.IpRanges[serviceName]

		return ok
	}

	return true
}

// pullIpRangesPerCloudProvider pull all ip ranges of all cloud providers
func (s *CloudProviderManager) pullIpRangesPerCloudProvider(cloudProviderName types.CloudProviderName) error {
	provider := s.cloudProviders[cloudProviderName]
	if provider == nil {
		return fmt.Errorf("cloud provider %s does not exist", cloudProviderName)
	}

	s.log.Debugw("pulling ip ranges from provider", "provider", cloudProviderName)

	unparsedIpRange, err := s.requestIpRangesFromProvider(provider.Conf.IPRangesURL)
	if err != nil {
		return err
	}

	if provider.IpRangeInfo != nil && !provider.CloudProvider.IsNewVersion(unparsedIpRange, *provider.IpRangeInfo) {
		s.log.Debugw("no changes of ip ranges", "provider", cloudProviderName)
		s.log.Debugw("finished pulling new version", "provider", cloudProviderName)

		return nil
	}

	newIpRangeInfo, err := provider.CloudProvider.ConvertToGenericIpRanges(unparsedIpRange)
	if err != nil {
		return err
	}

	// We compute this into a list to indicate to the user which services are available on error during disruption creation
	provider.IpRangeInfo = newIpRangeInfo
	for service := range newIpRangeInfo.IpRanges {
		provider.ServiceList = append(provider.ServiceList, service)
	}

	return nil
}

// requestIpRangesFromProvider launches a HTTP GET request to pull the ip range json file from a url
func (s *CloudProviderManager) requestIpRangesFromProvider(url string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(response.Body)
}

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
	IPRangeFileVersion          map[string]string // version per url
	IPRanges                    map[string][]string
	ServiceList                 []string // Makes the process of getting the services names easier
	Conf                        types.CloudProviderConfig
}

// CloudProviderIPRangeManager Methods to verify and transform a specifid ip ranges list from a provider
type CloudProviderIPRangeManager interface {
	IsNewVersion([]byte, string) bool
	ConvertToGenericIPRanges([]byte) (string, map[string][]string, error)
}

func New(log *zap.SugaredLogger, config types.CloudProviderConfigs) (*CloudServicesProvidersManager, error) {
	cloudProviderMap := map[types.CloudProviderName]*CloudServicesProvider{
		types.CloudProviderAWS: {
			CloudProviderIPRangeManager: aws.New(),
			Conf: types.CloudProviderConfig{
				IPRangesURL: []string{"https://ip-ranges.amazonaws.com/ip-ranges.json"},
			},
		},
		types.CloudProviderGCP: {
			CloudProviderIPRangeManager: gcp.New(),
			Conf: types.CloudProviderConfig{
				IPRangesURL: []string{
					"https://www.gstatic.com/ipranges/goog.json",  // General IP Ranges from Google, contains some API ip ranges
					"https://www.gstatic.com/ipranges/cloud.json", // GCP IP Ranges from Google
				},
			},
		},
	}

	pullInterval, err := time.ParseDuration(config.PullInterval)
	if err != nil {
		return nil, err
	}

	manager := &CloudServicesProvidersManager{
		cloudProviders:       cloudProviderMap,
		log:                  log,
		periodicPullInterval: pullInterval,
	}

	if err := manager.PullIPRanges(); err != nil {
		return nil, err
	}

	return manager, nil
}

// StartPeriodicPull go routine pulling every interval all ip ranges of all cloud providers set up.
func (s *CloudServicesProvidersManager) StartPeriodicPull() {
	s.log.Infow("starting periodic pull and parsing of the cloud provider ip ranges", "interval", s.periodicPullInterval)

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
		return nil, fmt.Errorf("cloud provider %s does not exist", cloudProviderName)
	}

	allIPRanges := s.cloudProviders[cloudProviderName].IPRanges
	toSendIPRanges := map[string][]string{}

	for _, serviceName := range serviceNames {
		// if user has imputed the same service twice, we verify
		if _, ok := toSendIPRanges[serviceName]; ok {
			continue
		}

		if _, ok := allIPRanges[serviceName]; !ok {
			return nil, fmt.Errorf("service %s from %s does not exist", serviceName, cloudProviderName)
		}

		toSendIPRanges[serviceName] = append(toSendIPRanges[serviceName], allIPRanges[serviceName]...)
	}

	return toSendIPRanges, nil
}

// GetServiceList return the list of services of a specific cloud provider. Mostly used in disruption creation validation
func (s *CloudServicesProvidersManager) GetServiceList(cloudProviderName types.CloudProviderName) []string {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil
	}

	return s.cloudProviders[cloudProviderName].ServiceList
}

// pullIPRangesPerCloudProvider pull all ip ranges of all cloud providers
func (s *CloudServicesProvidersManager) pullIPRangesPerCloudProvider(cloudProviderName types.CloudProviderName) error {
	provider := s.cloudProviders[cloudProviderName]
	if provider == nil {
		return fmt.Errorf("cloud provider %s does not exist", cloudProviderName)
	}

	s.log.Debugw("pulling ip ranges from provider", "provider", cloudProviderName)

	if provider.ServiceList == nil {
		provider.ServiceList = []string{}
	}

	if provider.IPRangeFileVersion == nil {
		provider.IPRangeFileVersion = map[string]string{}
	}

	toChangeIpRanges := false
	unparsedIpRangesPerURL := map[string][]byte{}

	// pull and compare with old version of every file
	for _, ipRangesURL := range provider.Conf.IPRangesURL {
		unparsedIPRanges, err := s.requestIPRangesFromProvider(ipRangesURL)
		if err != nil {
			return err
		}

		unparsedIpRangesPerURL[ipRangesURL] = unparsedIPRanges

		if provider.IPRangeFileVersion[ipRangesURL] == "" || provider.CloudProviderIPRangeManager.IsNewVersion(unparsedIPRanges, provider.IPRangeFileVersion[ipRangesURL]) {
			toChangeIpRanges = true
		}
	}

	if !toChangeIpRanges {
		s.log.Debugw("no changes of ip ranges", "provider", cloudProviderName)
		s.log.Debugw("finished pulling new version", "provider", cloudProviderName)

		return nil
	}

	// We reset the map
	provider.IPRanges = map[string][]string{}

	for ipRangesURL, unparsedIpRanges := range unparsedIpRangesPerURL {
		version, genericIPRangesPerService, err := provider.CloudProviderIPRangeManager.ConvertToGenericIPRanges(unparsedIpRanges)
		if err != nil {
			return err
		}

		provider.IPRangeFileVersion[ipRangesURL] = version

		for serviceName, ipRanges := range genericIPRangesPerService {
			if len(provider.IPRanges[serviceName]) == 0 {
				provider.IPRanges[serviceName] = ipRanges

				// We compute this into a list to indicate to the user which services are available on error during disruption creation
				provider.ServiceList = append(provider.ServiceList, serviceName)
			} else {
				provider.IPRanges[serviceName] = append(provider.IPRanges[serviceName], ipRanges...)
			}
		}
	}

	for serviceName, ipRanges := range provider.IPRanges {
		s.log.Debugf("%s: service %s has %d ip ranges", cloudProviderName, serviceName, len(ipRanges))
	}

	return nil
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

	return io.ReadAll(response.Body)
}

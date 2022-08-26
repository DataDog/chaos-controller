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
	IpRangeInfo   *types.CloudProviderIpRange
	ServiceList   []string
	Conf          types.CloudProviderConfig
}

// CloudProvider Describe CloudProvider's methods
type CloudProvider interface {
	// Check if the ip ranges pulled are newer than the one we already have
	IsNewVersion([]byte, types.CloudProviderIpRange) bool
	// From an unmarshalled json result of a provider to a generic ip range struct
	ConvertToGenericIpRanges([]byte) (*types.CloudProviderIpRange, error)
}

func New(log *zap.SugaredLogger, awsConfig types.CloudProviderConfig) CloudProviderManager {
	cloudProviderMap := map[types.CloudProviderName]*CloudProviderData{
		types.CloudProviderAWS: {
			CloudProvider: aws.New(),
			Conf:          awsConfig,
			IpRangeInfo:   nil,
			ServiceList:   nil,
		},
	}

	return CloudProviderManager{
		cloudProviders: cloudProviderMap,
		log:            log,
	}
}

func (s *CloudProviderManager) Run(interval time.Duration) {
	s.log.Debugw("starting periodic pull and parsing of the cloud provider ip ranges")

	s.PullIpRanges()

	go func() {
		for {
			select {
			case closed := <-s.close:
				if closed {
					return
				}
			case <-time.After(interval):
				s.PullIpRanges()
			}
		}
	}()
}

func (s *CloudProviderManager) StopPeriodicPull() {
	s.log.Debugw("closing periodic pull and parsing of the cloud provider ip ranges")

	s.close <- true
}

func (s *CloudProviderManager) PullIpRanges() {
	s.log.Infow("pull and parse of the cloud provider ip ranges")
	for cloudProviderName := range s.cloudProviders {
		s.PullIpRangesPerCloudProvider(cloudProviderName)
	}
	s.log.Infow("finished pull and parse of the cloud provider ip ranges")
}

func (s *CloudProviderManager) PullIpRangesPerCloudProvider(cloudProviderName types.CloudProviderName) {
	provider := s.cloudProviders[cloudProviderName]

	s.log.Debugw("pulling ip ranges from provider", "provider", cloudProviderName)

	unparsedIpRange, err := s.getIpRangesFromProvider(provider.Conf.IPRangesURL)
	if err != nil {
		s.log.Errorw("could not get the new ip ranges from provider", "err", err.Error(), "provider", cloudProviderName)
		return
	}

	if provider.IpRangeInfo != nil && !provider.CloudProvider.IsNewVersion(unparsedIpRange, *provider.IpRangeInfo) {
		s.log.Debugw("no changes of ip ranges", "provider", cloudProviderName)
		s.log.Debugw("finished pulling new version", "provider", cloudProviderName)

		return
	}

	newIpRangeInfo, err := provider.CloudProvider.ConvertToGenericIpRanges(unparsedIpRange)
	if err != nil {
		s.log.Errorw("could not get the new ip ranges from provider", "err", err.Error(), "provider", cloudProviderName)

		return
	}

	// We compute this into a list to indicate to the user which services are available on error during disruption creation
	provider.IpRangeInfo = newIpRangeInfo
	for service := range newIpRangeInfo.IpRanges {
		provider.ServiceList = append(provider.ServiceList, service)
	}
}

func (s *CloudProviderManager) GetIpRanges(cloudProviderName types.CloudProviderName, serviceName string) ([]string, error) {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil, fmt.Errorf("this cloud provider does not exist")
	}

	ipRangeInfo := s.cloudProviders[cloudProviderName].IpRangeInfo

	if serviceName != "" {
		return ipRangeInfo.IpRanges[serviceName], nil
	}

	allIpRanges := []string{}

	for _, ipRanges := range ipRangeInfo.IpRanges {
		allIpRanges = append(allIpRanges, ipRanges...)
	}

	return allIpRanges, nil
}

func (s *CloudProviderManager) GetServiceList(cloudProviderName types.CloudProviderName) []string {
	if s.cloudProviders[cloudProviderName] == nil {
		return nil
	}

	return s.cloudProviders[cloudProviderName].ServiceList
}

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

func (s *CloudProviderManager) getIpRangesFromProvider(url string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(response.Body)
}

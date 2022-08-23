package cloudservice

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice/aws"
	"github.com/DataDog/chaos-controller/cloudservice/types"
	"go.uber.org/zap"
)

// CloudProviderManager manager used to pull and parse any provider ip ranges per service
type CloudProviderManager struct {
	cloudProviders map[types.CloudProviderName]CloudProvider
	log            *zap.SugaredLogger
}

// CloudProvider Describe CloudProvider's methods
type CloudProvider interface {
	// Check if the ip ranges pulled are newer than the one we already have
	IsNewVersion([]byte, types.CloudProviderIpRange) bool
	// From an unmarshalled json result of a provider to a generic ip range struct
	ConvertToGenericIpRanges([]byte) (*types.CloudProviderIpRange, error)
	// Get the configuration of a specific cloud provider
	GetConf() types.CloudProviderConfig
}

func New(awsConfig types.CloudProviderConfig) CloudProviderManager {
	cloudProviderMap := map[types.CloudProviderName]CloudProvider{
		types.CloudProviderAWS: aws.New(awsConfig),
	}

	return CloudProviderManager{
		cloudProviders: cloudProviderMap,
	}
}

func (s *CloudProviderManager) PullAllIpRanges() {
	for cloudProviderName := range s.cloudProviders {
		s.PullIpRangesPerCloudProvider(cloudProviderName)
	}
}

func (s *CloudProviderManager) PullIpRangesPerCloudProvider(cloudProviderName types.CloudProviderName) {
	provider := s.cloudProviders[cloudProviderName]

	s.log.Infow("pulling ip ranges from provider", "provider", cloudProviderName)

	unparsedIpRange, err := s.getIpRangesFromProvider(provider.GetConf().IPRangesURL)
	if err != nil {
		s.log.Errorw("could not get the new ip ranges from provider", "err", err.Error(), "provider", cloudProviderName)
		return
	}

	oldIpRanges, err := s.getIpRangesFromFile(provider.GetConf().IPRangesPath)
	if err != nil {
		s.log.Errorw("could not get the ip ranges from file", "err", err.Error(), "provider", cloudProviderName)
		return
	}

	if !provider.IsNewVersion(unparsedIpRange, *oldIpRanges) {
		s.log.Infow("no changes of ip ranges", "provider", cloudProviderName)
		s.log.Infow("finished pulling new version", "provider", cloudProviderName)

		return
	}

	newIpRanges, err := provider.ConvertToGenericIpRanges(unparsedIpRange)
	if err != nil {
		s.log.Errorw("could not get the new ip ranges from provider", "err", err.Error(), "provider", cloudProviderName)

		return
	}

	if err := s.writeIpRangesToFile(provider.GetConf().IPRangesPath, newIpRanges); err != nil {
		s.log.Errorw("could not write the new ip ranges to file", "err", err.Error(), "provider", cloudProviderName)
	}
}

func (s *CloudProviderManager) GetIpRanges(cloudProviderName types.CloudProviderName, serviceName string) ([]string, error) {
	provider := s.cloudProviders[cloudProviderName]

	ipRanges, err := s.getIpRangesFromFile(provider.GetConf().IPRangesPath)
	if err != nil {
		return nil, err
	}

	if serviceName != "" {
		return ipRanges.IpRanges[serviceName], nil
	}

	allIpRanges := []string{}

	for _, ipRanges := range ipRanges.IpRanges {
		allIpRanges = append(allIpRanges, ipRanges...)
	}

	return allIpRanges, nil
}

func (s *CloudProviderManager) writeIpRangesToFile(filepath string, ipRanges *types.CloudProviderIpRange) error {
	data, err := json.Marshal(ipRanges)
	if err != nil {
		return err
	}

	// Create if it doesn't exist, truncate if it does
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	file.Close()
	return nil
}

func (s *CloudProviderManager) getIpRangesFromFile(filepath string) (*types.CloudProviderIpRange, error) {
	file, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	ipRanges := types.CloudProviderIpRange{}

	if err := json.Unmarshal(file, &ipRanges); err != nil {
		return nil, err
	}

	return &ipRanges, nil
}

func (s *CloudProviderManager) getIpRangesFromProvider(url string) ([]byte, error) {
	client := http.Client{
		Timeout: time.Second * 10,
	}

	response, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	unparsedIpRanges := []byte{}
	_, err = response.Body.Read(unparsedIpRanges)
	if err != nil {
		return nil, err
	}

	return unparsedIpRanges, nil
}

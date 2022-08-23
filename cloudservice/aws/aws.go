package aws

import (
	"encoding/json"
	"time"

	"github.com/DataDog/chaos-controller/cloudservice/types"
)

type CloudService struct {
	types.CloudProviderConfig
}

type AWSIpRange struct {
	IPPrefix           string `json:"ip_prefix"`
	Region             string `json:"region"`
	Service            string `json:"service"`
	NetworkBorderGroup string `json:"network_border_group"`
}

type AWSIpRanges struct {
	SyncToken  string       `json:"syncToken"`
	CreateDate time.Time    `json:"createDate"`
	Prefixes   []AWSIpRange `json:"prefixes"`
}

func New(conf types.CloudProviderConfig) *CloudService {
	return &CloudService{conf}
}

func (s *CloudService) GetConf() types.CloudProviderConfig {
	return s.CloudProviderConfig
}

func (s *CloudService) IsNewVersion(newIpRanges []byte, oldIpRanges types.CloudProviderIpRange) bool {
	ipRanges := AWSIpRanges{}
	if err := json.Unmarshal(newIpRanges, &ipRanges); err != nil {
		return false
	}

	return ipRanges.SyncToken != oldIpRanges.Version
}

func (s *CloudService) ConvertToGenericIpRanges(unparsedIpRanges []byte) (*types.CloudProviderIpRange, error) {
	ipRanges := AWSIpRanges{}
	if err := json.Unmarshal(unparsedIpRanges, &ipRanges); err != nil {
		return nil, err
	}

	genericIpRanges := types.CloudProviderIpRange{
		CloudProviderServiceName: types.CloudProviderAWS,
		IpRanges:                 make(map[string][]string),
	}

	for _, ipRange := range ipRanges.Prefixes {
		if len(genericIpRanges.IpRanges[ipRange.Service]) == 0 {
			genericIpRanges.IpRanges[ipRange.Service] = []string{}
		}
		genericIpRanges.IpRanges[ipRange.Service] = append(genericIpRanges.IpRanges[ipRange.Service], ipRange.IPPrefix)
	}

	return &genericIpRanges, nil
}

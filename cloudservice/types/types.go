package types

import (
	"go.uber.org/zap"
)

type CloudProviderName string

const (
	CloudProviderDatadog CloudProviderName = "Datadog"
	CloudProviderGCP     CloudProviderName = "GCP"
	CloudProviderAWS     CloudProviderName = "AWS"
)

type CloudProviderIpRange struct {
	Version                  string
	CloudProviderServiceName CloudProviderName
	IpRanges                 map[string][]string
}

type CloudProviderConfig struct {
	IPRangesURL  string
	IPRangesPath string
	Log          *zap.SugaredLogger
}

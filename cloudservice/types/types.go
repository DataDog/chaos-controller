package types

type CloudProviderName string

const (
	CloudProviderDatadog CloudProviderName = "Datadog"
	CloudProviderGCP     CloudProviderName = "GCP"
	CloudProviderAWS     CloudProviderName = "AWS"
)

// CloudProviderIpRangeInfo information related to the ip ranges pulled from a cloud provider
type CloudProviderIpRangeInfo struct {
	Version                  string
	CloudProviderServiceName CloudProviderName
	IpRanges                 map[string][]string
}

// CloudProviderConfig Single configuration for any cloud provider
type CloudProviderConfig struct {
	IPRangesURL string `json:"iprangesurl"`
}

// CloudProviderConfigs all cloud provider configurations for the manager
type CloudProviderConfigs struct {
	Aws CloudProviderConfig `json:"aws"`
}

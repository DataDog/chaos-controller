// Code generated by client-gen. DO NOT EDIT.

package internalversion

import (
	"net/http"
	"v1beta1/v1beta1/scheme"

	rest "k8s.io/client-go/rest"
)

type ChaosInterface interface {
	RESTClient() rest.Interface
	DisruptionsGetter
	DisruptionCronsGetter
}

// ChaosClient is used to interact with features provided by the chaos.datadoghq.com group.
type ChaosClient struct {
	restClient rest.Interface
}

func (c *ChaosClient) Disruptions(namespace string) DisruptionInterface {
	return newDisruptions(c, namespace)
}

func (c *ChaosClient) DisruptionCrons(namespace string) DisruptionCronInterface {
	return newDisruptionCrons(c, namespace)
}

// NewForConfig creates a new ChaosClient for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, httpClient),
// where httpClient was generated with rest.HTTPClientFor(c).
func NewForConfig(c *rest.Config) (*ChaosClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	httpClient, err := rest.HTTPClientFor(&config)
	if err != nil {
		return nil, err
	}
	return NewForConfigAndClient(&config, httpClient)
}

// NewForConfigAndClient creates a new ChaosClient for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config, h *http.Client) (*ChaosClient, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientForConfigAndClient(&config, h)
	if err != nil {
		return nil, err
	}
	return &ChaosClient{client}, nil
}

// NewForConfigOrDie creates a new ChaosClient for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *ChaosClient {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new ChaosClient for the given RESTClient.
func New(c rest.Interface) *ChaosClient {
	return &ChaosClient{c}
}

func setConfigDefaults(config *rest.Config) error {
	config.APIPath = "/apis"
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	if config.GroupVersion == nil || config.GroupVersion.Group != scheme.Scheme.PrioritizedVersionsForGroup("chaos.datadoghq.com")[0].Group {
		gv := scheme.Scheme.PrioritizedVersionsForGroup("chaos.datadoghq.com")[0]
		config.GroupVersion = &gv
	}
	config.NegotiatedSerializer = scheme.Codecs

	if config.QPS == 0 {
		config.QPS = 5
	}
	if config.Burst == 0 {
		config.Burst = 10
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *ChaosClient) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}

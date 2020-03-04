package metrics

import (
	"fmt"

	"github.com/DataDog/chaos-controller/metrics/datadog"
	"github.com/DataDog/chaos-controller/metrics/noop"
)

// Sink describes a metric sink
type Sink interface {
	EventWithTags(title, text string, tags []string)
	EventCleanFailure(containerID, uid string)
	EventInjectFailure(containerID, uid string)
	MetricInjected(containerID, uid string, succeed bool, tags []string)
	MetricRulesInjected(containerID, uid string, succeed bool, tags []string)
	MetricCleaned(containerID, uid string, succeed bool, tags []string)
}

// GetSink returns an initiated sink
func GetSink(name string) (Sink, error) {
	switch name {
	case "datadog":
		return datadog.New(), nil
	case "noop":
		return noop.New(), nil
	default:
		return nil, fmt.Errorf("unsupported metrics sink: %s", name)
	}
}

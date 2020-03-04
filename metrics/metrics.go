package metrics

import (
	"fmt"

	"github.com/DataDog/chaos-controller/metrics/datadog"
	"github.com/DataDog/chaos-controller/metrics/noop"
)

type MetricsSink interface {
	EventWithTags(title, text string, tags []string)
	EventCleanFailure(containerID, uid string)
	EventInjectFailure(containerID, uid string)
	MetricInjected(containerID, uid string, succeed bool)
	MetricRulesInjected(containerID, uid string, succeed bool)
	MetricCleaned(containerID, uid string, succeed bool)
}

// GetSink returns an initiated sink
func GetSink(name string) (MetricsSink, error) {
	switch name {
	case "datadog":
		return datadog.New(), nil
	case "noop":
		return noop.New(), nil
	default:
		return nil, fmt.Errorf("unsupported metrics sink: %s", name)
	}
}

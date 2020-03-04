package noop

import (
	"log"
)

// NoopSink ...
type NoopSink struct{}

// New ...
func New() *NoopSink {
	return &NoopSink{}
}

// EventWithTags creates a new event with the given title, text and tags and send it
func (n *NoopSink) EventWithTags(title, text string, tags []string) {
	log.Printf("%v %v %v", title, text, tags)
}

// EventCleanFailure sends an event to datadog specifying a failure clean fail
func (n *NoopSink) EventCleanFailure(containerID, uid string) {
	log.Printf("EventCleanFailure %v", containerID)
}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func (n *NoopSink) EventInjectFailure(containerID, uid string) {
	log.Printf("EventInjectFailure %v", containerID)
}

// MetricInjected increments the injected metric
func (n *NoopSink) MetricInjected(containerID, uid string, succeed bool, tags []string) {
	log.Printf("MetricInjected %v", containerID)
}

// MetricRulesInjected rules.increments the injected metric
func (n *NoopSink) MetricRulesInjected(containerID, uid string, succeed bool, tags []string) {
	log.Printf("MetricRulesInjected %v", containerID)
}

// MetricCleaned increments the cleaned metric
func (n *NoopSink) MetricCleaned(containerID, uid string, succeed bool, tags []string) {
	log.Printf("MetricCleaned %v", containerID)
}

package datadog

import (
	"os"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
)

const metricPrefix = "chaos.nfi"

var instance *statsd.Client
var once sync.Once

// GetInstance returns (and initializes if needed) an instance of the Datadog statsd client
func GetInstance() *statsd.Client {
	once.Do(func() {
		var err error
		url := os.Getenv("STATSD_URL")
		instance, err = statsd.New(url, statsd.WithTags([]string{"app:chaos-fi-controller"}))
		if err != nil {
			panic(err)
		}
	})
	return instance
}

// EventWithTags creates a new event with the given title, text and tags and send it
func EventWithTags(title, text string, tags []string) {
	_ = GetInstance().Event(&statsd.Event{
		Title: title,
		Text:  text,
		Tags:  tags,
	})
}

// EventCleanFailure sends an event to datadog specifying a failure clean fail
func EventCleanFailure(containerID, uid string) {
	EventWithTags("network failure clean failed", "please check the cleanup pod logs to have more details",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
	)
}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func EventInjectFailure(containerID, uid string) {
	EventWithTags("network failure injection failed", "please check the inject pod logs to have more details",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
	)
}

func metricWithStatus(name, containerID, uid string, succeed bool) {
	var status string
	if succeed {
		status = "succeed"
	} else {
		status = "failed"
	}

	_ = GetInstance().Incr(name, []string{"containerID" + containerID, "UID:" + uid, "status:" + status}, 1)
}

// MetricInjected increments the injected metric
func MetricInjected(containerID, uid string, succeed bool) {
	metricWithStatus(metricPrefix+".injected", containerID, uid, succeed)
}

// MetricRulesInjected rules.increments the injected metric
func MetricRulesInjected(containerID, uid string, succeed bool) {
	metricWithStatus(metricPrefix+".rules.injected", containerID, uid, succeed)
}

// MetricCleaned increments the cleaned metric
func MetricCleaned(containerID, uid string, succeed bool) {
	metricWithStatus(metricPrefix+".cleaned", containerID, uid, succeed)
}

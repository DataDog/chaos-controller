package datadog

import (
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"os"
	"sync"
	"time"
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
func EventWithTags(title, text, host string, tags []string, timestamp *time.Time) {
	e := statsd.Event{
		Title: title,
		Text:  text,
		Tags:  tags,
	}
	if len(host) > 0 {
		e.Hostname = host
	}
	if timestamp != nil {
		e.Timestamp = *timestamp
	}
	GetInstance().Event(&e)
}

//EventNewPod creates a new event specifying the name of a newly created pod
func EventNewPod(podType, failure, instance, host string, timestamp *time.Time, tags []string) {
	EventWithTags(fmt.Sprintf("New %s %s Pod", podType, failure), fmt.Sprintf("Created cleanup %s for %s \"%s\"", podType, failure, instance), host,
		tags,
		timestamp,
	)
}

// EventCleanFailure sends an event to datadog specifying a failure clean fail
func EventCleanFailure(containerID, uid string) {
	EventWithTags("network failure clean failed", "please check the cleanup pod logs to have more details", "",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
		nil,
	)
}

// EventInjectFailure sends an event to datadog specifying a failure inject fail
func EventInjectFailure(containerID, uid string) {
	EventWithTags("network failure injection failed", "please check the inject pod logs to have more details", "",
		[]string{
			"containerID:" + containerID,
			"UID:" + uid,
		},
		nil,
	)
}

func metricWithStatus(name, containerID, uid string, succeed bool) {
	var status string
	if succeed {
		status = "succeed"
	} else {
		status = "failed"
	}

	GetInstance().Incr(name, []string{"containerID" + containerID, "UID:" + uid, "status:" + status}, 1)
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

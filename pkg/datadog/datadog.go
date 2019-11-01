package datadog

import (
	"github.com/DataDog/datadog-go/statsd"
	"os"
	"sync"
	"time"
)

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

// Creates new Event with given parameters
func NewEvent(title, text, hostname string, tags []string, timestamp time.Time) *statsd.Event {
	var event *statsd.Event
	event = &statsd.Event{
		Title:     title,
		Text:      text,
		Hostname:  hostname,
		Tags:      tags,
		Timestamp: timestamp,
	}

	return event
}

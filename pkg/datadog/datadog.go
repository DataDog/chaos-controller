package datadog

import (
	"os"
	"sync"

	"github.com/DataDog/datadog-go/statsd"
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

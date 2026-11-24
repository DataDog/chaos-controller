// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cloudservice_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/DataDog/chaos-controller/cloudservice"
	"github.com/DataDog/chaos-controller/cloudservice/types"
	cLog "github.com/DataDog/chaos-controller/log"
	"go.uber.org/zap"
)

type extraRangeRoundTripper struct{}

func (extraRangeRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"syncToken":"1","prefixes":[{"ipv4Prefix":"34.80.0.0/15"}]}`)),
		Request:    request,
	}, nil
}

func TestCloudExtraRangesRejectMalformedEntries(t *testing.T) {
	for _, entry := range []string{
		"Google",
		"",
		";199.36.153.8/30",
		"Google;",
		"Google;199.36.153.8/30;",
		"Google;199.36.153.8/30;;199.36.153.4/30",
		"Google; ",
	} {
		t.Run(entry, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("malformed extra range %q panicked instead of returning an error: %v", entry, recovered)
				}
			}()

			ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
			manager, err := cloudservice.New(ctx, types.CloudProviderConfigs{
				GCP: types.CloudProviderConfig{
					Enabled:       true,
					IPRangesURL:   "https://ranges.invalid/goog.json",
					ExtraIPRanges: []string{entry},
				},
			}, &http.Client{Transport: extraRangeRoundTripper{}})
			if err == nil {
				ranges, lookupErr := manager.GetServicesIPRanges(ctx, types.CloudProviderGCP, []string{"Google"})
				t.Fatalf("accepted malformed extra range %q: ranges=%v, lookup error=%v", entry, ranges, lookupErr)
			}
		})
	}
}

func TestCloudExtraRangesPreserveValidEntries(t *testing.T) {
	ctx := cLog.WithLogger(context.Background(), zap.NewNop().Sugar())
	manager, err := cloudservice.New(ctx, types.CloudProviderConfigs{
		GCP: types.CloudProviderConfig{
			Enabled:       true,
			IPRangesURL:   "https://ranges.invalid/goog.json",
			ExtraIPRanges: []string{"Google;199.36.153.8/30;199.36.153.4/30"},
		},
	}, &http.Client{Transport: extraRangeRoundTripper{}})
	if err != nil {
		t.Fatal(err)
	}

	ranges, err := manager.GetServicesIPRanges(ctx, types.CloudProviderGCP, []string{"Google"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"34.80.0.0/15", "199.36.153.8/30", "199.36.153.4/30"}
	if !reflect.DeepEqual(ranges["Google"], want) {
		t.Fatalf("Google ranges = %v; want %v", ranges["Google"], want)
	}
}

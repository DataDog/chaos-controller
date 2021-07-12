// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package validation

import "reflect"

// DDValidationMarker is the interface for Validation Markers, which apply rules to a given structure's field
type DDValidationMarker interface {
	// ApplyRule asserts the marker's rule is checked and returns an error if it isn't (invalidating the config)
	ApplyRule(reflect.Value) error
}

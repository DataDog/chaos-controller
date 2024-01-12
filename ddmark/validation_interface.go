// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package ddmark

import "reflect"

// DDValidationMarker is the interface for Validation Markers, which apply rules to a given structure's field
type DDValidationMarker interface {
	// ApplyRule asserts the marker's rule is checked and returns an error if it isn't (invalidating the config)
	ApplyRule(reflect.Value) error
	// ValueCheckError returns a marker's error message for an incorrect value/presence check
	ValueCheckError() error
	// TypeCheckError returns a marker's error message for an incorrect type check
	TypeCheckError(reflect.Value) error
}

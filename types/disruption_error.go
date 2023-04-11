// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package types

type DisruptionError struct {
	context map[string]string
	Err     error
}

func (d DisruptionError) Error() string {
	return d.Err.Error()
}

func (d DisruptionError) Context() map[string]string {
	if d.context == nil {
		return map[string]string{}
	}

	return d.context
}

func (d DisruptionError) AddContext(key string, value string) {
	if d.context == nil {
		d.context = map[string]string{}
	}

	d.context[key] = value
}

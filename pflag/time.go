// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package pflag

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
)

type timeWithFormat struct {
	format string
	inner  *time.Time
}

// NewTimeWithFormat will create a new cobra plag that will update provided time using provided format when a flag is set through a command line
func NewTimeWithFormat(format string, v *time.Time) (pflag.Value, error) {
	if v == nil {
		return nil, fmt.Errorf("given time must not be nil")
	}

	t := timeWithFormat{
		format,
		v,
	}

	return &t, nil
}

// String return the string representation of the time with format
// it will be either an empty time formatted is Set was not called yet
// or the time formatted in the desired format is Set was already called
func (t *timeWithFormat) String() string {
	return t.inner.Format(t.format)
}

// Set defines underlying time to provided string that should be in stored format
// it will be called by cobra on command line parsing
func (t *timeWithFormat) Set(v string) error {
	tv, err := time.Parse(t.format, v)
	if err != nil {
		return fmt.Errorf("unable to parse %s to '%s' time format: %w", v, t.format, err)
	}

	*t.inner = tv

	return nil
}

// Type provide the unique type name associated with this plag.Value implementation
// here it's a timeformat-FORMAT generated string
func (t *timeWithFormat) Type() string {
	return fmt.Sprintf("timeformat-%s", t.format)
}

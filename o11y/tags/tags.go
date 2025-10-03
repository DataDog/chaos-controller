// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package tags

import "fmt"

// FormatTag formats a tag with key:value format for metrics and observability
func FormatTag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}

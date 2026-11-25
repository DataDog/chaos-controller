// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package watchers

import "fmt"

// TargetResourceIndexKey is the name of the controller-runtime cache field index
// used to look up DisruptionRollouts by their target resource.
const TargetResourceIndexKey = "targetResource"

// TargetResourceIndexedValue builds the cache index key for the target resource
// of a DisruptionRollout. The separator cannot appear in Kubernetes kinds,
// namespaces or names, so distinct target resources always map to distinct keys.
func TargetResourceIndexedValue(kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

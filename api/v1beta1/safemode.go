// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

// SafemodeSpec represents a spec with a switch turning safemode on and all safety net switches to turn off
// IgnoreAll, by default, is defined by the cluster the controller is running on
// If the use is running on a cluster that has IgnoreAll as false, they will have to manually ignore the other safety nets they do not want considered
// If IgnoreAll is true by default, the user can change its value
type SafemodeSpec struct {
	IgnoreAll                 bool `json:"ignoreAll,omitempty"`
	IgnoreCountNotTooLarge    bool `json:"ignoreCountNotToLarge,omitempty"`
	IgnoreNoPortOrHost        bool `json:"ignoreNoPortOrHost,omitempty"`
	IgnoreSporadicTargets     bool `json:"ignoreSporadicTargets,omitempty"`
	IgnoreSpecificContainDisk bool `json:"ignoreSpecificContainDisk,omitempty"`
}

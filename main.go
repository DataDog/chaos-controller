// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package main

import (
	"github.com/DataDog/chaos-controller/manager"
)

func main() {
	manager.StartManager(nil)
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package types

type NotifiersConfig struct {
	Common notifiersCommonConfig `json:"notifiersCommonConfig"`
	Noop   notifierNoopConfig    `json:"notifierNoopConfig"`
	Slack  notifierSlackConfig   `json:"notifierSlackConfig"`
}

type notifiersCommonConfig struct {
	ClusterName string
}

type notifierSlackConfig struct {
	Enabled  bool
	Filepath string
}

type notifierNoopConfig struct {
	Enabled bool
}

type NotifierDriver string

const (
	// NotifierDriverSlack is the Slack driver
	NotifierDriverSlack NotifierDriver = "slack"

	// NotifierDriverNoop is a noop driver mainly used for testing
	NotifierDriverNoop NotifierDriver = "noop"
)

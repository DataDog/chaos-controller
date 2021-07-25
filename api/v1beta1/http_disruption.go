// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
//	"errors"
	"fmt"
	"strings"
)

// HTTPDisruptionSpec represents an http disruption
type HTTPDisruptionSpec []HTTPDisruption

type HTTPDisruption struct {
	Domains []TargetDomain `json:"domains"`
	Ports   []int          `json:"ports"`
}

// TargetDomain represents an http-hosting endpoint
type TargetDomain struct {
	Domain string         `json:"domain"`
	Header []RequestField `json:"header"`
}

// RequestFields represents some of the fields present in the http header
type RequestField struct {
	Uri    string `json:"uri"`
	Method string `json:"method"`
}

// GenerateArgs generates injection pod arguments for the given spec
func (s HTTPDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"http-disruption",
	}

	arg := "--port-list "

	for _, port := range s[0].Ports {
		arg = fmt.Sprintf("%s,%d", arg, port)
	}
	args = append(args, arg)

	targetDomainArgs := []string{}

	for _, target := range s[0].Domains {
		whiteSpaceCleanedDomain := strings.ReplaceAll(target.Domain, " ", "")
		for _, header := range(target.Header) {
			whiteSpaceCleanedUri := strings.ReplaceAll(header.Uri, " ", "")
			whiteSpaceCleanedMethod := strings.ReplaceAll(header.Method, " ", "")
			arg = fmt.Sprintf("%s%s;%s", whiteSpaceCleanedDomain, whiteSpaceCleanedUri, whiteSpaceCleanedMethod)
			targetDomainArgs = append(targetDomainArgs, arg)
		}
	}

	args = append(args, "--request-field")

	// Each value passed to --request-field should be of the form `uri;method`, e.g.
	// `/foo/bar/baz;GET`
	args = append(args, strings.Split(strings.Join(targetDomainArgs, " --request-field "), " ")...)

	return args
}

// Validate validates that there are no missing URIs or methods in the given http disruption spec
func (s HTTPDisruptionSpec) Validate() error {
	// TODO: implement
	return nil
}

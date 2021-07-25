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
type HTTPDisruptionSpec []TargetDomain

// TargetDomain represents an http-hosting endpoint
type TargetDomain struct {
	Domain string         `json:"domain"`
	Header []RequestField `json:"header"`
	Ports PortTypes `json:"ports"`
}

// RequestFields represents some of the fields present in the http request header
type RequestField struct {
	Uri    string `json:"uri"`
	Method string `json:"method"`
}

// PortTypes represents the HTTPS and HTTP ports to forward via iptables and the ones that the corresponding proxy services should listen on
type PortTypes struct {
	HTTPS []int `json:"https"`
	HTTP []int `json:"http"`
}

// GenerateArgs generates injection pod arguments for the given spec
func (s HTTPDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"http-disruption",
	}

	httpPortArg := "--http-port-list "
	httpsPortArg := " --https-port-list "

	targetDomainArgs := []string{}

	for _, target := range s {
		whiteSpaceCleanedDomain := strings.ReplaceAll(target.Domain, " ", "")

		for _, header := range(target.Header) {
			whiteSpaceCleanedUri := strings.ReplaceAll(header.Uri, " ", "")
			whiteSpaceCleanedMethod := strings.ReplaceAll(header.Method, " ", "")
			arg := fmt.Sprintf("--request-field %s%s;%s", whiteSpaceCleanedDomain, whiteSpaceCleanedUri, whiteSpaceCleanedMethod)
			targetDomainArgs = append(targetDomainArgs, arg)
			}

		for _, port := range target.Ports.HTTP {
			httpPortArg = fmt.Sprintf("%s,%d", httpPortArg, port)
		}
		for _, port := range target.Ports.HTTPS {
			httpsPortArg = fmt.Sprintf("%s,%d", httpsPortArg, port)
		}
	}

	args = append(args, httpPortArg)
	args = append(args, httpsPortArg)

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

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"strings"
)

// HTTPDisruptionSpec represents an http disruption
type HTTPDisruptionSpec []TargetDomain

// TargetDomain represents an http-hosting endpoint
type TargetDomain struct {
	Domain string         `json:"domain"`
	Header []RequestField `json:"header"`
	Ports  PortTypes      `json:"ports"`
}

// RequestFields represents some of the fields present in the http request header
type RequestField struct {
	Uri    string `json:"uri"`
	Method string `json:"method"`
}

// PortTypes represents the HTTPS and HTTP ports to forward via iptables and the ones that the corresponding proxy services should listen on
type PortTypes struct {
	HTTPS []string `json:"https"`
	HTTP  []string `json:"http"`
}

// GenerateArgs generates injection pod arguments for the given spec
func (s HTTPDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"http-disruption",
	}

	httpPortArg := "--http-port-list"
	httpsPortArg := "--https-port-list"

	targetDomainArgs := []string{}

	for _, target := range s {
		whiteSpaceCleanedDomain := strings.ReplaceAll(target.Domain, " ", "")

		for _, header := range target.Header {
			whiteSpaceCleanedUri := strings.ReplaceAll(header.Uri, " ", "")
			whiteSpaceCleanedMethod := strings.ReplaceAll(header.Method, " ", "")
			arg := fmt.Sprintf("%s;%s;%s", whiteSpaceCleanedDomain, whiteSpaceCleanedUri, whiteSpaceCleanedMethod)
			targetDomainArgs = append(targetDomainArgs, arg)
		}

		httpPortArg = fmt.Sprintf("%s=%s", httpPortArg, strings.Join(target.Ports.HTTP, ","))
		httpsPortArg = fmt.Sprintf("%s=%s", httpsPortArg, strings.Join(target.Ports.HTTPS, ","))
	}

	args = append(args, httpPortArg)
	args = append(args, httpsPortArg)

	// Each value passed to --request-field should be of the form `domain+uri;method`, e.g.
	// `foo.com/bar/baz;GET`
	args = append(args, fmt.Sprintf("--request-field=%s", strings.Join(targetDomainArgs, ",")))

	return args
}

// Validate validates that there are no missing domains, ports, URIs or methods in the given http disruption spec
func (s HTTPDisruptionSpec) Validate() error {
	for _, target := range s {
		if target.Domain == "" {
			return errors.New("No domain specified in the http disruption")
		}

		if len(target.Header) == 0 {
			return errors.New("No header values specified in the http disruption")
		}

		for _, header := range target.Header {
			if header.Uri == "" {
				return errors.New("Header is missing a uri value in http disruption")
			}
			if header.Method == "" {
				return errors.New("Header is missing an http method value in http disruption")
			}
		}

		if len(target.Ports.HTTP) == 0 {
			return errors.New("Missing http ports in the http disruption")
		}

		if len(target.Ports.HTTPS) == 0 {
			return errors.New("Missing https ports in the http disruption")
		}
	}

	return nil
}

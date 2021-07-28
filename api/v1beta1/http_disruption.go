// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2021 Datadog, Inc.

package v1beta1

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// HTTPDisruptionSpec represents an http disruption
type HTTPDisruptionSpec struct {
	Domains []TargetDomain `json:"domains,omitempty"`
	// +nullable
	HttpPorts []int `json:"httpPorts,omitempty"`
	// +nullable
	HttpsPorts []int `json:"httpsPorts,omitempty"`
}

// TargetDomain represents an http-hosting endpoint
type TargetDomain struct {
	Domain string         `json:"domain,omitempty"`
	Header []RequestField `json:"header,omitempty"`
}

// RequestFields represents some of the fields present in the http request header
type RequestField struct {
	Uri    string `json:"uri"`
	Method string `json:"method"`
}

// GenerateArgs generates injection pod arguments for the given spec
func (s HTTPDisruptionSpec) GenerateArgs() []string {
	args := []string{
		"http-disruption",
	}

	targetDomainArgs := []string{}
	httpPortArgs := []string{}
	httpsPortArgs := []string{}

	for _, port := range s.HttpPorts {
		httpPortArgs = append(httpPortArgs, strconv.Itoa(port))
	}
	for _, port := range s.HttpsPorts {
		httpsPortArgs = append(httpsPortArgs, strconv.Itoa(port))
	}

	for _, target := range s.Domains {
		whiteSpaceCleanedDomain := strings.ReplaceAll(target.Domain, " ", "")

		for _, header := range target.Header {
			whiteSpaceCleanedUri := strings.ReplaceAll(header.Uri, " ", "")
			whiteSpaceCleanedMethod := strings.ReplaceAll(header.Method, " ", "")
			arg := fmt.Sprintf("%s;%s;%s", whiteSpaceCleanedDomain, whiteSpaceCleanedUri, whiteSpaceCleanedMethod)
			targetDomainArgs = append(targetDomainArgs, arg)
		}
	}

	// Each value passed to --http-port-list and --https-port-list should be a single port
	args = append(args, "--http-port-list")

	args = append(args, strings.Split(strings.Join(httpPortArgs, " --http-port-list "), " ")...)
	args = append(args, "--https-port-list")
	args = append(args, strings.Split(strings.Join(httpsPortArgs, " --https-port-list "), " ")...)

	// Each value passed to --request-field should be of the form `domain;uri;method`, e.g.
	// `foo.com;/bar/baz;GET`
	args = append(args, "--request-field")
	args = append(args, strings.Split(strings.Join(targetDomainArgs, " --request-field "), " ")...)

	return args
}

// Validate validates that there are no missing domains, ports, URIs or methods in the given http disruption spec
func (s HTTPDisruptionSpec) Validate() error {
	for _, target := range s.Domains {
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
	}
	for _, port := range s.HttpPorts {
		if port < 1 || port > 65535 {
			return errors.New("HTTP port is out of range in http disruption")
		}
	}
	for _, port := range s.HttpsPorts {
		if port < 1 || port > 65535 {
			return errors.New("HTTPS port is out of range in http disruption")
		}
	}

	return nil
}

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package http

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/tidwall/gjson"

	cLog "github.com/DataDog/chaos-controller/log"
	"github.com/DataDog/chaos-controller/o11y/tags"
)

// BearerAuthTokenProvider defines an interface for retrieving authentication tokens.
type BearerAuthTokenProvider interface {
	AuthToken(ctx context.Context) (string, error)
}

// ensure bearerAuthTokenProvider implements BearerAuthTokenProvider interface.
var _ BearerAuthTokenProvider = bearerAuthTokenProvider{}

// bearerAuthTokenProvider implements the BearerAuthTokenProvider interface for
// retrieving bearer tokens from a specified URL using HTTP GET request.
type bearerAuthTokenProvider struct {
	URL       string
	Client    *http.Client
	Headers   map[string]string
	TokenPath string
}

// NewBearerAuthTokenProvider creates a new BearerAuthTokenProvider.
func NewBearerAuthTokenProvider(client *http.Client, url string, headers map[string]string, tokenPath string) BearerAuthTokenProvider {
	return bearerAuthTokenProvider{
		url,
		client,
		headers,
		tokenPath,
	}
}

// AuthToken retrieves an authentication token from a remote server using an HTTP GET request.
func (b bearerAuthTokenProvider) AuthToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.URL, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create http request for URL %s: %w", b.URL, err)
	}

	for headerKey, headerValue := range b.Headers {
		req.Header.Add(headerKey, headerValue)
	}

	logger := cLog.FromContext(ctx).With(tags.ReqKey, fmt.Sprintf("%+v", req))

	logger.Debugw("sending request to get token")

	res, err := b.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to do http request to get token: %w", err)
	}

	defer func() {
		if err = res.Body.Close(); err != nil {
			logger.Warnw("an error occurred while closing body after reading auth token", tags.ErrorKey, err)
		}
	}()

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return "", fmt.Errorf("received response contains unexpected status code %d when retrieving auth", res.StatusCode)
	}

	tokenBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error when reading token: %w", err)
	}

	if b.TokenPath == "" {
		return string(tokenBytes), nil
	}

	value := gjson.Get(string(tokenBytes), b.TokenPath)
	if value.Exists() {
		return value.String(), nil
	}

	return "", fmt.Errorf("auth response body does not contains expected token path %s", b.TokenPath)
}

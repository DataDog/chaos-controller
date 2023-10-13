// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package http_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	. "github.com/DataDog/chaos-controller/eventnotifier/http"
	"github.com/DataDog/chaos-controller/mocks"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var _ = Describe("Auth", func() {
	var (
		logger               *zap.SugaredLogger
		httpRoundTripperMock *mocks.RoundTripperMock
		httpClient           *http.Client
		headers              map[string]string
		authTokenProvider    BearerAuthTokenProvider
	)

	BeforeEach(func() {
		logger = zaptest.NewLogger(GinkgoT()).Sugar()
		httpRoundTripperMock = mocks.NewRoundTripperMock(GinkgoT())
		headers = make(map[string]string)

		httpClient = &http.Client{
			Transport: httpRoundTripperMock,
		}

		authTokenProvider = nil
	})

	DescribeTable("returns expected errors",
		func(ctx SpecContext, url, tokenPath string, httpResponse *http.Response, httpError error, expected any) {
			if httpResponse != nil || httpError != nil {
				httpRoundTripperMock.EXPECT().RoundTrip(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
					return httpResponse, httpError
				}).Once()
			}

			authTokenProvider = NewBearerAuthTokenProvider(logger, httpClient, url, headers, tokenPath)

			token, err := authTokenProvider.AuthToken(ctx)

			Expect(err).To(MatchError(expected))
			Expect(token).To(BeEmpty())
		},
		Entry("invalid url returns error", ":", "", nil, nil, `unable to create http request for URL :: parse ":": missing protocol scheme`),
		Entry("server error returns error", "", "", nil, errors.New("server error"), `unable to do http request to get token: Get "": server error`),
		Entry("3xx status code returns error", "", "", &http.Response{StatusCode: http.StatusContinue}, nil, "received response contains unexpected status code 100 when retrieving auth"),
		Entry("3xx status code returns error", "", "", &http.Response{StatusCode: http.StatusMovedPermanently}, nil, "received response contains unexpected status code 301 when retrieving auth"),
		Entry("4xx status code returns error", "", "", &http.Response{StatusCode: http.StatusNotFound}, nil, "received response contains unexpected status code 404 when retrieving auth"),
		Entry("5xx status code returns error", "", "", &http.Response{StatusCode: http.StatusInternalServerError}, nil, "received response contains unexpected status code 500 when retrieving auth"),
		Entry("2xx status no auth path returns error", "", "some.path", &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte(`{}`)))}, nil, "auth response body does not contains expected token path some.path"),
	)

	It("returns a token from a valid response body", func(ctx SpecContext) {
		authTokenProvider = NewBearerAuthTokenProvider(logger, httpClient, "", headers, "some.path")

		httpRoundTripperMock.EXPECT().RoundTrip(mock.Anything).RunAndReturn(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader([]byte(`{"some": {"path": "value"}}`)))}, nil
		}).Once()

		token, err := authTokenProvider.AuthToken(ctx)

		Expect(err).ToNot(HaveOccurred())
		Expect(token).To(Equal("value"))
	})
})

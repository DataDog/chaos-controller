// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package api_test

import (
	"testing"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/ddmark"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var ddMarkClient ddmark.Client

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

var _ = BeforeSuite(func() {
	var err error
	ddMarkClient, err = ddmark.NewClient(v1beta1.EmbeddedChaosAPI)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(ddMarkClient.CleanupLibraries()).ToNot(HaveOccurred())
})

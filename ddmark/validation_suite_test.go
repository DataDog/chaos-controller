// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package ddmark_test

import (
	"github.com/DataDog/chaos-controller/ddmark"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestValidationTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValidationTest Suite")
}

var _ddmark ddmark.Ddmark

var _ = BeforeSuite(func() {
	_ddmark = ddmark.NewDdmark()
	ddmark.InitLibrary(ddmark.EmbeddedDDMarkAPI, "ddmark-api")
})

var _ = AfterSuite(func() {
	ddmark.CleanupLibraries("ddmark-api")
})

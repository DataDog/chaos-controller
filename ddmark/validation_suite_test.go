// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2024 Datadog, Inc.

package ddmark_test

import (
	"fmt"
	"testing"

	"github.com/DataDog/chaos-controller/ddmark"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValidationTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ValidationTest Suite")
}

var client ddmark.Client

var _ = BeforeSuite(func() {
	var err error
	client, err = ddmark.NewClient(ddmark.EmbeddedDDMarkAPI)
	if err != nil {
		fmt.Println("error setting up ddmark")
	}
})

var _ = AfterSuite(func() {
	Expect(client.CleanupLibraries()).To(Succeed())
})

// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"encoding/json"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PodReplacementSpec", func() {
	It("preserves an explicit false deleteStorage value when marshaling", func() {
		data, err := json.Marshal(v1beta1.PodReplacementSpec{DeleteStorage: false})
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(MatchJSON(`{"deleteStorage":false}`))
	})
})

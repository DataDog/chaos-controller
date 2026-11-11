// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package network

import (
	"os"
	"path/filepath"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var _ = Describe("example manifests match the disruption API", func() {
	It("decodes and validates the complete reference's network section", func() {
		body, err := os.ReadFile("../examples/complete.yaml")
		Expect(err).ToNot(HaveOccurred())
		var disruption v1beta1.Disruption
		Expect(yaml.UnmarshalStrict(body, &disruption)).To(Succeed())
		Expect(disruption.Spec.Network).ToNot(BeNil())
		Expect(disruption.Spec.Network.Services).To(HaveLen(2))
		Expect(disruption.Spec.Network.Services[1].Ports).To(Equal([]v1beta1.NetworkDisruptionServicePortSpec{
			{Port: 8080},
			{Port: 8081},
		}))

		// The reference deliberately lists mutually exclusive disruption kinds.
		// Validate its network section independently rather than claiming the
		// combined reference is an admissible experiment.
		networkSpec := v1beta1.DisruptionSpec{
			Level:    disruption.Spec.Level,
			Count:    disruption.Spec.Count,
			Selector: disruption.Spec.Selector,
			Network:  disruption.Spec.Network,
		}
		Expect(networkSpec.Validate()).To(Succeed())
	})

	It("validates the standalone disruption examples", func() {
		paths, err := filepath.Glob("../examples/*.yaml")
		Expect(err).ToNot(HaveOccurred())
		Expect(paths).ToNot(BeEmpty())
		for _, path := range paths {
			if filepath.Base(path) == "complete.yaml" {
				continue // The combined reference is checked separately above.
			}
			body, err := os.ReadFile(path)
			Expect(err).ToNot(HaveOccurred(), path)
			var metadata metav1.TypeMeta
			Expect(yaml.Unmarshal(body, &metadata)).To(Succeed(), path)
			if metadata.Kind != "Disruption" {
				continue
			}
			var disruption v1beta1.Disruption
			Expect(yaml.UnmarshalStrict(body, &disruption)).To(Succeed(), path)
			Expect(disruption.Spec.Validate()).To(Succeed(), path)
		}
	})
})

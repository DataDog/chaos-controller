// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

var _ = Describe("gRPC override argument boundaries", func() {
	It("preserves whitespace inside an empty-object override as one flag value", func() {
		spec := v1beta1.GRPCDisruptionSpec{
			Port: 50051,
			Endpoints: []v1beta1.EndpointAlteration{
				{TargetEndpoint: "/svc/Method", OverrideToReturn: "{ }", QueryPercent: 100},
				{TargetEndpoint: "/svc/Other", ErrorToReturn: "UNAVAILABLE", QueryPercent: 25},
			},
		}
		Expect(spec.Validate()).To(Succeed())
		flags := pflag.NewFlagSet("grpc-disruption", pflag.ContinueOnError)
		flags.Int("port", 0, "")
		flags.StringArray("endpoint-alterations", nil, "")
		args := spec.GenerateArgs()
		Expect(flags.Parse(args[1:])).To(Succeed())
		alterations, err := flags.GetStringArray("endpoint-alterations")
		Expect(err).NotTo(HaveOccurred())
		Expect(alterations).To(Equal([]string{
			"/svc/Method;override;{ };100",
			"/svc/Other;error;UNAVAILABLE;25",
		}))
		Expect(flags.Args()).To(BeEmpty())
	})
})

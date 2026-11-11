// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package injector

import (
	"errors"
	"os"
	"time"

	"github.com/DataDog/chaos-controller/api"
	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/netns"
	"github.com/DataDog/chaos-controller/network"
	"github.com/DataDog/chaos-controller/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

var _ = Describe("network jitter percentage precision", func() {
	assertJitter := func(spec v1beta1.NetworkDisruptionSpec, expected time.Duration) {
		count := intstr.FromInt(1)
		input := v1beta1.DisruptionSpec{
			Level:    types.DisruptionLevelNode,
			Count:    &count,
			Selector: labels.Set{"app": "target"},
			Network:  &spec,
		}
		Expect(input.Validate()).To(Succeed())

		namespaces := netns.NewManagerMock(GinkgoT())
		namespaces.EXPECT().Enter().Return(nil).Once()
		namespaces.EXPECT().Exit().Return(nil).Maybe()
		links := network.NewNetlinkAdapterMock(GinkgoT())
		stop := errors.New("jitter test stops before interface configuration")
		links.EXPECT().LinkList(false, mock.Anything).Return(nil, stop).Once()
		controller := network.NewTrafficControllerMock(GinkgoT())
		controller.EXPECT().AddNetem([]string{"test0"}, "root", "", time.Duration(spec.Delay)*time.Millisecond, expected, spec.Drop, spec.Corrupt, spec.Duplicate).Return(nil).Once()

		inj := networkDisruptionInjector{
			spec: spec,
			config: NetworkDisruptionInjectorConfig{
				Config: Config{
					Log:        zap.NewNop().Sugar(),
					Netns:      namespaces,
					Disruption: api.DisruptionArgs{Level: types.DisruptionLevelNode},
				},
				NetlinkAdapter:    links,
				TrafficController: controller,
			},
		}
		// Inject performs the production calculation before interface discovery.
		// Stop there, then exercise the generated operation without touching a network.
		Expect(inj.Inject()).To(MatchError(ContainSubstring(stop.Error())))
		Expect(inj.operations).To(HaveLen(1))
		Expect(inj.operations[0]([]string{"test0"}, "root", "")).To(Succeed())
	}

	DescribeTable("preserves the requested percentage",
		func(delay, percent uint, expected time.Duration) {
			assertJitter(v1beta1.NetworkDisruptionSpec{Delay: delay, DelayJitter: percent}, expected)
		},
		Entry("default ten percent of one millisecond", uint(1), uint(0), 100*time.Microsecond),
		Entry("one percent of one millisecond", uint(1), uint(1), 10*time.Microsecond),
		Entry("fractional milliseconds", uint(15), uint(10), 1500*time.Microsecond),
		Entry("integer percentage without floating-point truncation", uint(100), uint(29), 29*time.Millisecond),
		Entry("maximum admitted delay and percentage", uint(60000), uint(100), 60*time.Second),
	)

	It("honors the documented default jitter in the node-delay example", func() {
		body, err := os.ReadFile("../examples/network_delay_node.yaml")
		Expect(err).ToNot(HaveOccurred())
		var disruption v1beta1.Disruption
		Expect(yaml.UnmarshalStrict(body, &disruption)).To(Succeed())
		Expect(disruption.Spec.Validate()).To(Succeed())
		Expect(disruption.Spec.Network).ToNot(BeNil())
		assertJitter(*disruption.Spec.Network, 100*time.Microsecond)
	})
})

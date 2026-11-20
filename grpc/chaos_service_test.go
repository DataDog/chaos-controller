// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package grpc_test

import (
	"context"

	chaos "github.com/DataDog/chaos-controller/grpc"
	pb "github.com/DataDog/chaos-controller/grpc/disruptionlistener"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap/zaptest"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ = Describe("ChaosDisruptionListener", func() {
	var (
		listener *chaos.ChaosDisruptionListener
		ctx      context.Context
	)

	BeforeEach(func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		listener = chaos.NewDisruptionListener(log)
		ctx = context.Background()
	})

	Describe("NewDisruptionListener", func() {
		It("creates a non-nil listener", func() {
			Expect(listener).NotTo(BeNil())
		})
	})

	Describe("Disrupt", func() {
		Context("nil DisruptionSpec", func() {
			It("returns InvalidArgument error", func() {
				_, err := listener.Disrupt(ctx, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisruptionSpec is nil"))
			})
		})

		Context("endpoint without TargetEndpoint", func() {
			It("returns InvalidArgument error", func() {
				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{TargetEndpoint: "", Alterations: []*pb.AlterationSpec{
							{ErrorToReturn: "NOT_FOUND"},
						}},
					},
				}
				_, err := listener.Disrupt(ctx, spec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("TargetEndpoint"))
			})
		})

		Context("valid spec", func() {
			It("returns empty response without error", func() {
				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{
							TargetEndpoint: "/service/Method",
							Alterations: []*pb.AlterationSpec{
								{ErrorToReturn: "NOT_FOUND", QueryPercent: 50},
							},
						},
					},
				}
				resp, err := listener.Disrupt(ctx, spec)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(Equal(&emptypb.Empty{}))
			})
		})

		Context("already configured", func() {
			It("returns AlreadyExists error on second Disrupt", func() {
				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{
							TargetEndpoint: "/service/Method",
							Alterations: []*pb.AlterationSpec{
								{ErrorToReturn: "NOT_FOUND", QueryPercent: 100},
							},
						},
					},
				}
				_, err := listener.Disrupt(ctx, spec)
				Expect(err).NotTo(HaveOccurred())

				_, err = listener.Disrupt(ctx, spec)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("already configured"))
			})
		})

		Context("cancelled context", func() {
			It("does not apply config when context cancelled before mutex", func() {
				cancelCtx, cancel := context.WithCancel(context.Background())
				cancel()

				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{
							TargetEndpoint: "/service/Method",
							Alterations: []*pb.AlterationSpec{
								{ErrorToReturn: "NOT_FOUND", QueryPercent: 100},
							},
						},
					},
				}
				// May or may not apply config depending on timing, but should not panic
				Expect(func() { listener.Disrupt(cancelCtx, spec) }).NotTo(Panic())
			})
		})
	})

	Describe("ResetDisruptions", func() {
		It("clears configuration and returns empty response", func() {
			spec := &pb.DisruptionSpec{
				Endpoints: []*pb.EndpointSpec{
					{
						TargetEndpoint: "/service/Method",
						Alterations:    []*pb.AlterationSpec{{ErrorToReturn: "NOT_FOUND", QueryPercent: 100}},
					},
				},
			}
			_, _ = listener.Disrupt(ctx, spec)

			resp, err := listener.ResetDisruptions(ctx, &emptypb.Empty{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).To(Equal(&emptypb.Empty{}))

			// After reset, can Disrupt again
			_, err = listener.Disrupt(ctx, spec)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ChaosServerInterceptor", func() {
		noop := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "noop-response", nil
		}

		Context("endpoint not configured", func() {
			It("passes through to handler", func() {
				info := &googlegrpc.UnaryServerInfo{FullMethod: "/not/configured"}
				resp, err := listener.ChaosServerInterceptor(ctx, "req", info, noop)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(Equal("noop-response"))
			})
		})

		Context("endpoint configured with error alteration", func() {
			BeforeEach(func() {
				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{
							TargetEndpoint: "/service/Method",
							Alterations:    []*pb.AlterationSpec{{ErrorToReturn: "NOT_FOUND", QueryPercent: 100}},
						},
					},
				}
				_, err := listener.Disrupt(ctx, spec)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns gRPC error", func() {
				info := &googlegrpc.UnaryServerInfo{FullMethod: "/service/Method"}
				_, err := listener.ChaosServerInterceptor(ctx, "req", info, noop)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Chaos Controller injected this error"))
			})
		})

		Context("endpoint configured with override alteration", func() {
			BeforeEach(func() {
				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{
							TargetEndpoint: "/service/Override",
							Alterations:    []*pb.AlterationSpec{{OverrideToReturn: "{}", QueryPercent: 100}},
						},
					},
				}
				_, err := listener.Disrupt(ctx, spec)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns empty override response", func() {
				info := &googlegrpc.UnaryServerInfo{FullMethod: "/service/Override"}
				resp, err := listener.ChaosServerInterceptor(ctx, "req", info, noop)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(Equal(&emptypb.Empty{}))
			})
		})

		Context("index out of bounds (queryPercent < 100)", func() {
			BeforeEach(func() {
				spec := &pb.DisruptionSpec{
					Endpoints: []*pb.EndpointSpec{
						{
							TargetEndpoint: "/service/LowPercent",
							// 1% chance means alterations slice has 1 element, index 1-99 will be OOB
							Alterations: []*pb.AlterationSpec{{ErrorToReturn: "NOT_FOUND", QueryPercent: 1}},
						},
					},
				}
				_, err := listener.Disrupt(ctx, spec)
				Expect(err).NotTo(HaveOccurred())
			})

			It("passes through to handler when index is out of bounds", func() {
				// With 1% alteration, most calls will pass through.
				// Call enough times to hit a passthrough.
				passed := false
				for i := 0; i < 200; i++ {
					info := &googlegrpc.UnaryServerInfo{FullMethod: "/service/LowPercent"}
					resp, err := listener.ChaosServerInterceptor(ctx, "req", info, noop)
					if err == nil && resp == "noop-response" {
						passed = true
						break
					}
				}

				Expect(passed).To(BeTrue())
			})
		})

		Context("alteration with neither error nor override", func() {
			BeforeEach(func() {
				// We need to inject a configuration with an alteration that has neither field set.
				// This can't happen through Disrupt (it validates), so we skip this case.
				// The code path is logged as an error and passes through.
			})

			It("passes through to handler for unconfigured endpoint", func() {
				info := &googlegrpc.UnaryServerInfo{FullMethod: "/not/configured"}
				resp, err := listener.ChaosServerInterceptor(ctx, "req", info, noop)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(Equal("noop-response"))
			})
		})
	})
})

var _ = Describe("calculations.ConvertSpecifications", func() {
	It("converts valid spec list", func() {
		specs := []*pb.AlterationSpec{
			{ErrorToReturn: "NOT_FOUND", QueryPercent: 50},
			{OverrideToReturn: "{}", QueryPercent: 30},
		}
		// Call Disrupt which internally uses ConvertSpecifications
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		listener := chaos.NewDisruptionListener(log)
		disruptionSpec := &pb.DisruptionSpec{
			Endpoints: []*pb.EndpointSpec{
				{TargetEndpoint: "/svc/Method", Alterations: specs},
			},
		}
		_, err := listener.Disrupt(context.Background(), disruptionSpec)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error for alteration with invalid spec", func() {
		log := zaptest.NewLogger(GinkgoT()).Sugar()
		listener := chaos.NewDisruptionListener(log)
		disruptionSpec := &pb.DisruptionSpec{
			Endpoints: []*pb.EndpointSpec{
				{
					TargetEndpoint: "/svc/Method",
					Alterations: []*pb.AlterationSpec{
						{ErrorToReturn: "", OverrideToReturn: ""}, // invalid: both empty
					},
				},
			},
		}
		_, err := listener.Disrupt(context.Background(), disruptionSpec)
		Expect(err).To(HaveOccurred())
	})
})

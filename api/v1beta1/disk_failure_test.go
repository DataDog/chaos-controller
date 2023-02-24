// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.

package v1beta1_test

import (
	"github.com/DataDog/chaos-controller/api/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("DiskFailureSpec", func() {
	When("Call the 'Validate' method", func() {
		var path string
		var err error

		JustBeforeEach(func() {
			df := v1beta1.DiskFailureSpec{
				Path: path,
			}
			err = df.Validate()
		})

		Context("with a valid path not exceeding 62 characters", func() {
			BeforeEach(func() {
				path = randStringRunes(rand.IntnRange(1, 62))
			})
			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("with a valid path", func() {
			BeforeEach(func() {
				path = "   " + randStringRunes(rand.IntnRange(61, 62)) + "   "
			})
			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
		})

		Context("with a path more than 62 characters", func() {
			BeforeEach(func() {
				path = randStringRunes(rand.IntnRange(63, 10000))
			})
			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("the path of the disk failure disruption must not exceed 62 characters"))
			})
		})

		Context("with an empty path", func() {
			BeforeEach(func() {
				path = ""
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("the path of the disk failure disruption must not be empty"))
			})
		})

		Context("with a blank path", func() {
			BeforeEach(func() {
				path = "   "
			})

			It("should return an error", func() {
				Expect(err).Should(HaveOccurred())
				Expect(err.Error()).Should(Equal("the path of the disk failure disruption must not be blank"))
			})
		})
	})

	When("Call the 'GenerateArgs' method", func() {
		var args []string
		var path string

		JustBeforeEach(func() {
			diskFailureSpec := v1beta1.DiskFailureSpec{Path: path}
			args = diskFailureSpec.GenerateArgs()
		})

		Context("with a '/' path", func() {
			BeforeEach(func() {
				path = "/"
			})
			It("should return valid args", func() {
				Expect(args).Should(Equal([]string{"disk-failure", "--path", "/"}))
			})
		})

		Context("with a '/sub/path/'", func() {
			BeforeEach(func() {
				path = "/sub/path/"
			})
			It("should return valid args", func() {
				Expect(args).Should(Equal([]string{"disk-failure", "--path", "/sub/path/"}))
			})
		})

		Context("with a path containing spaces", func() {
			BeforeEach(func() {
				path = "  /  "
			})
			It("should return args with the path without spaces", func() {
				Expect(args).Should(Equal([]string{"disk-failure", "--path", "/"}))
			})
		})
	})
})

func randStringRunes(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

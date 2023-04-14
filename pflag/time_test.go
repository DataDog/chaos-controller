// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023 Datadog, Inc.
package pflag_test

import (
	"time"

	"github.com/DataDog/chaos-controller/pflag"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NewTimeWithFormat expectations", func() {
	const (
		someTime                         = "2023-05-10T15:28:28+02:00"
		someInvalidTime                  = "223-05-10T15:28:28+02:00"
		noNewMatchError, noSetMatchError = "", ""
	)

	DescribeTable(
		"Time",
		func(format string, set string, setMatchError string) {
			v := time.Time{}
			sut, err := pflag.NewTimeWithFormat(format, &v)

			By("creating a new plag with expected error")
			Expect(err).To(Succeed())

			By("confirming created flag is not nil")
			Expect(sut).ToNot(BeNil())

			By("confirming default value is empty value")
			Expect(sut.String()).To(Equal(time.Time{}.Format(format)))

			By("confirming type is uniq due to format inclusion in it")
			Expect(sut.Type()).To(ContainSubstring(format))

			if setMatchError == "" {
				By("confirming set succeeded")
				Expect(sut.Set(set)).To(Succeed())
			} else {
				By("confirming set returned expected error")
				Expect(sut.Set(set)).To(MatchError(setMatchError))
			}

			By("confirming set value is related to provided format")
			Expect(sut.String()).To(Equal(v.Format(format)))
		},
		Entry("RFC3339 format expectectations", time.RFC3339, someTime, noSetMatchError),
		Entry("invalid time returns error", time.RFC3339, someInvalidTime, `unable to parse 223-05-10T15:28:28+02:00 to '2006-01-02T15:04:05Z07:00' time format: parsing time "223-05-10T15:28:28+02:00" as "2006-01-02T15:04:05Z07:00": cannot parse "223-05-10T15:28:28+02:00" as "2006"`),
		Entry("invalid format returns error", "whatever", someTime, `unable to parse 2023-05-10T15:28:28+02:00 to 'whatever' time format: parsing time "2023-05-10T15:28:28+02:00" as "whatever": cannot parse "2023-05-10T15:28:28+02:00" as "whatever"`),
	)
})

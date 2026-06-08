// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2026 Datadog, Inc.

package cpuset_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-controller/cpuset"
)

var _ = Describe("Builder", func() {
	It("builds empty set when no elements added", func() {
		b := cpuset.NewBuilder()
		s := b.Result()
		Expect(s.Size()).To(Equal(0))
		Expect(s.IsEmpty()).To(BeTrue())
	})

	It("builds set from added elements", func() {
		b := cpuset.NewBuilder()
		b.Add(1, 2, 3)
		s := b.Result()
		Expect(s.Size()).To(Equal(3))
		Expect(s.Contains(1)).To(BeTrue())
		Expect(s.Contains(2)).To(BeTrue())
		Expect(s.Contains(3)).To(BeTrue())
	})

	It("deduplicates elements", func() {
		b := cpuset.NewBuilder()
		b.Add(1, 1, 2)
		Expect(b.Result().Size()).To(Equal(2))
	})
})

var _ = Describe("NewCPUSet", func() {
	It("creates empty set with no args", func() {
		s := cpuset.NewCPUSet()
		Expect(s.IsEmpty()).To(BeTrue())
	})

	It("creates set with single element", func() {
		s := cpuset.NewCPUSet(5)
		Expect(s.Size()).To(Equal(1))
		Expect(s.Contains(5)).To(BeTrue())
	})

	It("creates set with multiple elements", func() {
		s := cpuset.NewCPUSet(1, 2, 3)
		Expect(s.Size()).To(Equal(3))
	})

	It("deduplicates elements", func() {
		s := cpuset.NewCPUSet(1, 1, 2)
		Expect(s.Size()).To(Equal(2))
	})
})

var _ = Describe("Contains", func() {
	It("returns true for present element", func() {
		Expect(cpuset.NewCPUSet(1, 2, 3).Contains(2)).To(BeTrue())
	})

	It("returns false for absent element", func() {
		Expect(cpuset.NewCPUSet(1, 2, 3).Contains(9)).To(BeFalse())
	})
})

var _ = Describe("Equals", func() {
	It("equal sets", func() {
		Expect(cpuset.NewCPUSet(1, 2).Equals(cpuset.NewCPUSet(2, 1))).To(BeTrue())
	})

	It("different sets", func() {
		Expect(cpuset.NewCPUSet(1, 2).Equals(cpuset.NewCPUSet(1, 3))).To(BeFalse())
	})

	It("two empty sets are equal", func() {
		Expect(cpuset.NewCPUSet().Equals(cpuset.NewCPUSet())).To(BeTrue())
	})

	It("empty vs non-empty", func() {
		Expect(cpuset.NewCPUSet().Equals(cpuset.NewCPUSet(1))).To(BeFalse())
	})
})

var _ = Describe("Filter", func() {
	s := cpuset.NewCPUSet(1, 2, 3, 4)

	It("keeps elements matching predicate", func() {
		even := s.Filter(func(n int) bool { return n%2 == 0 })
		Expect(even.Equals(cpuset.NewCPUSet(2, 4))).To(BeTrue())
	})

	It("always-false predicate returns empty set", func() {
		Expect(s.Filter(func(int) bool { return false }).IsEmpty()).To(BeTrue())
	})

	It("always-true predicate returns full set", func() {
		Expect(s.Filter(func(int) bool { return true }).Equals(s)).To(BeTrue())
	})
})

var _ = Describe("FilterNot", func() {
	s := cpuset.NewCPUSet(1, 2, 3, 4)

	It("removes elements matching predicate", func() {
		odd := s.FilterNot(func(n int) bool { return n%2 == 0 })
		Expect(odd.Equals(cpuset.NewCPUSet(1, 3))).To(BeTrue())
	})

	It("always-true predicate returns empty set", func() {
		Expect(s.FilterNot(func(int) bool { return true }).IsEmpty()).To(BeTrue())
	})

	It("always-false predicate returns full set", func() {
		Expect(s.FilterNot(func(int) bool { return false }).Equals(s)).To(BeTrue())
	})
})

var _ = Describe("IsSubsetOf", func() {
	It("empty set is subset of any set", func() {
		Expect(cpuset.NewCPUSet().IsSubsetOf(cpuset.NewCPUSet(1, 2))).To(BeTrue())
	})

	It("proper subset", func() {
		Expect(cpuset.NewCPUSet(1, 2).IsSubsetOf(cpuset.NewCPUSet(1, 2, 3))).To(BeTrue())
	})

	It("equal sets are subsets of each other", func() {
		s := cpuset.NewCPUSet(1, 2)
		Expect(s.IsSubsetOf(s)).To(BeTrue())
	})

	It("not a subset when elements missing", func() {
		Expect(cpuset.NewCPUSet(1, 4).IsSubsetOf(cpuset.NewCPUSet(1, 2, 3))).To(BeFalse())
	})
})

var _ = Describe("Union", func() {
	It("disjoint sets", func() {
		result := cpuset.NewCPUSet(1, 2).Union(cpuset.NewCPUSet(3, 4))
		Expect(result.Equals(cpuset.NewCPUSet(1, 2, 3, 4))).To(BeTrue())
	})

	It("overlapping sets deduplicate", func() {
		result := cpuset.NewCPUSet(1, 2, 3).Union(cpuset.NewCPUSet(2, 3, 4))
		Expect(result.Equals(cpuset.NewCPUSet(1, 2, 3, 4))).To(BeTrue())
	})

	It("union with empty returns original", func() {
		s := cpuset.NewCPUSet(1, 2)
		Expect(s.Union(cpuset.NewCPUSet()).Equals(s)).To(BeTrue())
	})
})

var _ = Describe("UnionAll", func() {
	It("combines multiple sets", func() {
		s := cpuset.NewCPUSet(1)
		result := s.UnionAll([]cpuset.CPUSet{cpuset.NewCPUSet(2, 3), cpuset.NewCPUSet(4)})
		Expect(result.Equals(cpuset.NewCPUSet(1, 2, 3, 4))).To(BeTrue())
	})

	It("empty slice returns copy of receiver", func() {
		s := cpuset.NewCPUSet(1, 2)
		Expect(s.UnionAll([]cpuset.CPUSet{}).Equals(s)).To(BeTrue())
	})
})

var _ = Describe("Intersection", func() {
	It("returns common elements", func() {
		result := cpuset.NewCPUSet(1, 2, 3).Intersection(cpuset.NewCPUSet(2, 3, 4))
		Expect(result.Equals(cpuset.NewCPUSet(2, 3))).To(BeTrue())
	})

	It("disjoint sets produce empty intersection", func() {
		result := cpuset.NewCPUSet(1, 2).Intersection(cpuset.NewCPUSet(3, 4))
		Expect(result.IsEmpty()).To(BeTrue())
	})
})

var _ = Describe("Difference", func() {
	It("returns elements in s1 not in s2", func() {
		result := cpuset.NewCPUSet(1, 2, 3).Difference(cpuset.NewCPUSet(2))
		Expect(result.Equals(cpuset.NewCPUSet(1, 3))).To(BeTrue())
	})

	It("difference with empty returns original", func() {
		s := cpuset.NewCPUSet(1, 2)
		Expect(s.Difference(cpuset.NewCPUSet()).Equals(s)).To(BeTrue())
	})

	It("difference with self returns empty", func() {
		s := cpuset.NewCPUSet(1, 2)
		Expect(s.Difference(s).IsEmpty()).To(BeTrue())
	})
})

var _ = Describe("ToSlice", func() {
	It("returns sorted slice", func() {
		s := cpuset.NewCPUSet(3, 1, 2)
		Expect(s.ToSlice()).To(Equal([]int{1, 2, 3}))
	})

	It("empty set returns empty slice", func() {
		Expect(cpuset.NewCPUSet().ToSlice()).To(BeEmpty())
	})
})

var _ = Describe("ToSliceNoSort", func() {
	It("contains all elements (unsorted)", func() {
		s := cpuset.NewCPUSet(3, 1, 2)
		Expect(s.ToSliceNoSort()).To(ConsistOf(1, 2, 3))
	})

	It("empty set returns empty slice", func() {
		Expect(cpuset.NewCPUSet().ToSliceNoSort()).To(BeEmpty())
	})
})

var _ = Describe("String", func() {
	DescribeTable("formats as Linux CPU list",
		func(cpus []int, expected string) {
			s := cpuset.NewCPUSet(cpus...)
			Expect(s.String()).To(Equal(expected))
		},
		Entry("empty set", []int{}, ""),
		Entry("single CPU", []int{5}, "5"),
		Entry("consecutive range", []int{0, 1, 2, 3, 4, 5}, "0-5"),
		Entry("non-consecutive", []int{0, 2, 4}, "0,2,4"),
		Entry("mixed", []int{0, 1, 2, 5, 7, 8, 9}, "0-2,5,7-9"),
	)
})

var _ = Describe("Clone", func() {
	It("clone equals original", func() {
		s := cpuset.NewCPUSet(1, 2, 3)
		Expect(s.Clone().Equals(s)).To(BeTrue())
	})

	It("clone of empty set is empty", func() {
		Expect(cpuset.NewCPUSet().Clone().IsEmpty()).To(BeTrue())
	})
})

var _ = Describe("MustParse", func() {
	It("parses valid input", func() {
		s := cpuset.MustParse("0-3,5")
		Expect(s.Contains(0)).To(BeTrue())
		Expect(s.Contains(3)).To(BeTrue())
		Expect(s.Contains(5)).To(BeTrue())
		Expect(s.Size()).To(Equal(5))
	})
})

var _ = Describe("Parse", func() {
	DescribeTable("success cases",
		func(input string, expected []int) {
			s, err := cpuset.Parse(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Equals(cpuset.NewCPUSet(expected...))).To(BeTrue())
		},
		Entry("empty string → empty set", "", []int{}),
		Entry("single CPU", "0", []int{0}),
		Entry("range", "0-3", []int{0, 1, 2, 3}),
		Entry("mixed", "0-2,5,7-9", []int{0, 1, 2, 5, 7, 8, 9}),
		Entry("single comma-separated list", "1,2,3", []int{1, 2, 3}),
	)

	DescribeTable("error cases",
		func(input string) {
			_, err := cpuset.Parse(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("non-numeric", "abc"),
		Entry("empty range segment", "0,,1"),
		Entry("reversed range", "5-3"),
		Entry("too many dashes", "1-2-3"),
		Entry("leading dash", "-5"),
		Entry("trailing dash", "5-"),
		Entry("negative CPU", "-1"),
		Entry("CPU >= 8192", "8192"),
		Entry("range too large", "0-8192"),
	)
})

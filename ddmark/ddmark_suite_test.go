package ddmark_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDdmark(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DDmark Suite")
}

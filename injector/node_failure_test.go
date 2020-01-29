package injector_test

import (
	"os"
	"reflect"

	"bou.ke/monkey"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/DataDog/chaos-fi-controller/api/v1beta1"
	. "github.com/DataDog/chaos-fi-controller/injector"
)

var _ = Describe("Failure", func() {
	var f NodeFailureInjector
	var osOpenFilePath, osWriteStringValue []string
	BeforeEach(func() {
		f = NodeFailureInjector{
			Injector: Injector{
				UID: "fake",
			},
			Spec: &v1beta1.NodeFailureSpec{},
		}

		// os
		var file *os.File
		monkey.Patch(os.OpenFile, func(path string, _ int, _ os.FileMode) (*os.File, error) {
			osOpenFilePath = append(osOpenFilePath, path)
			return nil, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(file), "WriteString", func(_ *os.File, data string) (int, error) {
			osWriteStringValue = append(osWriteStringValue, data)
			return 0, nil
		})
		monkey.PatchInstanceMethod(reflect.TypeOf(file), "Close", func(*os.File) error {
			return nil
		})
	})

	AfterEach(func() {
		monkey.UnpatchAll()
	})

	Describe("injection", func() {
		It("should write to the sysrq file", func() {
			f.Inject()
			Expect(len(osOpenFilePath)).To(Equal(2))
			Expect(len(osWriteStringValue)).To(Equal(2))
			Expect(osOpenFilePath[0]).To(Equal("/mnt/sysrq"))
			Expect(osWriteStringValue[0]).To(Equal("1"))
			Expect(osOpenFilePath[1]).To(Equal("/mnt/sysrq-trigger"))
			Expect(osWriteStringValue[1]).To(Equal("c"))
		})
	})
})

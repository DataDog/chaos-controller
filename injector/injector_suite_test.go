package injector_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

var _ = BeforeSuite(func() {
	z, _ := zap.NewDevelopment()
	log = z.Sugar()
})

func TestInjector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Injector Suite")
}

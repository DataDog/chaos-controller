module github.com/DataDog/chaos-fi-controller

go 1.13

require (
	bou.ke/monkey v1.0.2
	github.com/DataDog/datadog-go v3.3.1+incompatible
	github.com/Microsoft/hcsshim v0.8.7 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/containerd/containerd v1.3.2
	github.com/containerd/continuity v0.0.0-20200107062522-0ec596719c75 // indirect
	github.com/containerd/fifo v0.0.0-20191213151349-ff969a566b00 // indirect
	github.com/containerd/ttrpc v0.0.0-20191028202541-4f1b8fe65a5c // indirect
	github.com/containerd/typeurl v0.0.0-20190911142611-5eb25027c9fd // indirect
	github.com/coreos/go-iptables v0.4.5
	github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/go-logr/logr v0.1.0
	github.com/gogo/googleapis v1.3.1 // indirect
	github.com/gogo/protobuf v1.3.0
	github.com/miekg/dns v1.1.27
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v0.1.1 // indirect
	github.com/opencontainers/runtime-spec v1.0.1
	github.com/prometheus/common v0.0.0-20181126121408-4724e9255275
	github.com/spf13/cobra v0.0.5
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df
	go.etcd.io/bbolt v1.3.3 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478
	gopkg.in/alecthomas/kingpin.v2 v2.2.6 // indirect
	k8s.io/api v0.0.0-20190918155943-95b840bb6a1f
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)

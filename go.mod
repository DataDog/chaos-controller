module github.com/DataDog/chaos-controller

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.2.12
	github.com/DataDog/datadog-go v4.0.0+incompatible
	github.com/Microsoft/hcsshim v0.8.9 // indirect
	github.com/Microsoft/hcsshim/test v0.0.0-20200818230740-94556e86d3db // indirect
	github.com/avast/retry-go v2.6.0+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/containerd/cgroups v0.0.0-20200817152742-7a3c009711fb // indirect
	github.com/containerd/containerd v1.4.8
	github.com/containerd/continuity v0.0.0-20200710164510-efbc4488d8fe // indirect
	github.com/containerd/fifo v0.0.0-20200410184934-f15a3290365b // indirect
	github.com/containerd/go-runc v0.0.0-20200707131846-23d84c510c41 // indirect
	github.com/containerd/ttrpc v1.0.1 // indirect
	github.com/containerd/typeurl v1.0.1 // indirect
	github.com/coreos/go-iptables v0.5.0
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.7.3-0.20190327010347-be7ac8be2ae0
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/golang/protobuf v1.4.3
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/miekg/dns v1.1.31
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/opencontainers/runc v1.0.0-rc95 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1
	github.com/vishvananda/netlink v1.1.0
	github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae
	go.etcd.io/bbolt v1.3.5 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/sys v0.0.0-20210511113859-b0526f3d8744
	golang.org/x/text v0.3.6 // indirect
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.25.0
	gotest.tools/v3 v3.0.2 // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	k8s.io/kubernetes v1.13.0
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/yaml v1.2.0
)

module github.com/DataDog/chaos-controller

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.3.2
	github.com/DataDog/datadog-go v4.8.2+incompatible
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/containerd/containerd v1.5.9
	github.com/coreos/go-iptables v0.6.0
	github.com/docker/docker v17.12.0-ce-rc1.0.20200916142827-bd33bbf0497b+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/ghodss/yaml v1.0.0
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/markbates/pkger v0.17.1
	github.com/miekg/dns v1.1.25
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/opencontainers/runc v1.0.3
	github.com/slack-go/slack v0.9.5
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/vishvananda/netlink v1.1.1-0.20201029203352-d40f9887b852
	github.com/vishvananda/netns v0.0.0-20200728191858-db3c7e526aae
	go.uber.org/zap v1.19.0
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023
	golang.org/x/sys v0.0.0-20210817190340-bfb29a6856f2
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog/v2 v2.9.0
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/controller-tools v0.7.0
	sigs.k8s.io/yaml v1.2.0
)

module github.com/DataDog/chaos-controller

go 1.23.1

require (
	github.com/AlecAivazis/survey/v2 v2.3.6
	github.com/DataDog/datadog-go v4.8.3+incompatible
	github.com/DataDog/jsonapi v0.8.6
	github.com/aquasecurity/libbpfgo v0.5.1-libbpf-1.2
	github.com/aquasecurity/libbpfgo/helpers v0.4.5
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/containerd/containerd v1.6.26
	github.com/coreos/go-iptables v0.6.0
	github.com/docker/docker v26.1.5+incompatible
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-logr/logr v1.4.2
	github.com/go-logr/zapr v1.3.0
	github.com/go-playground/locales v0.14.1
	github.com/go-playground/universal-translator v0.18.1
	github.com/go-playground/validator/v10 v10.23.0
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/miekg/dns v1.1.55
	github.com/mitchellh/go-homedir v1.1.0
	github.com/onsi/ginkgo/v2 v2.23.4
	github.com/onsi/gomega v1.36.3
	github.com/opencontainers/runc v1.1.15
	github.com/slack-go/slack v0.12.2
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.15.0
	github.com/stretchr/testify v1.9.0
	github.com/tidwall/gjson v1.16.0
	github.com/vishvananda/netlink v1.2.1-beta.2.0.20230420174744-55c8b9515a01
	github.com/vishvananda/netns v0.0.5-0.20230405050519-16c2fa0b2f57
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/trace v1.29.0
	go.uber.org/zap v1.27.0
	golang.org/x/sys v0.32.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/DataDog/dd-trace-go.v1 v1.70.1
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.29.3
	k8s.io/apimachinery v0.29.3
	k8s.io/cli-runtime v0.26.1
	k8s.io/client-go v0.29.3
	k8s.io/klog/v2 v2.120.1
	sigs.k8s.io/controller-runtime v0.17.2
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/DataDog/datadog-agent/pkg/proto v0.58.0 // indirect
	github.com/DataDog/datadog-agent/pkg/trace v0.58.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/log v0.58.0 // indirect
	github.com/DataDog/datadog-agent/pkg/util/scrubber v0.58.0 // indirect
	github.com/DataDog/go-libddwaf/v3 v3.5.1 // indirect
	github.com/DataDog/go-runtime-metrics-internal v0.0.0-20241106155157-194426bbbd59 // indirect
	github.com/DataDog/go-sqllexer v0.0.14 // indirect
	github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes v0.20.0 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/eapache/queue/v2 v2.0.0-20230407133247-75960ed334e4 // indirect
	github.com/ebitengine/purego v0.6.0-alpha.5 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.3 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.7 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20220913051719-115f729f3c8c // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/signal v0.6.0 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/power-devops/perfstat v0.0.0-20220216144756-c35f1ee13d7c // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/shirou/gopsutil/v3 v3.24.4 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/collector/component v0.104.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.104.0 // indirect
	go.opentelemetry.io/collector/pdata v1.11.0 // indirect
	go.opentelemetry.io/collector/semconv v0.104.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.29.0 // indirect
	go.opentelemetry.io/otel/sdk v1.29.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/exp v0.0.0-20240318143956-a85f2c67cd81 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240822170219-fc7c04adadcd // indirect
)

require (
	github.com/DataDog/appsec-internal-go v1.9.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.58.0 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.58.0 // indirect
	github.com/DataDog/datadog-go/v5 v5.5.0 // indirect
	github.com/DataDog/go-tuf v1.1.0-0.5.2 // indirect
	github.com/DataDog/gostackparse v0.7.0 // indirect
	github.com/DataDog/sketches-go v1.4.5 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.9.10 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cilium/ebpf v0.9.1 // indirect
	github.com/containerd/cgroups v1.1.0 // indirect
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/containerd/fifo v1.1.0 // indirect
	github.com/containerd/ttrpc v1.2.2 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/cyphar/filepath-securejoin v0.2.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.0 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/klauspost/compress v1.17.1 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/mapstructure v1.5.1-0.20231216201459-8508981c8b6c // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc2.0.20221005185240-3a7f492d3f1b // indirect
	github.com/opencontainers/runtime-spec v1.1.0-rc.3 // indirect
	github.com/opencontainers/selinux v1.11.0 // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/pelletier/go-toml/v2 v2.0.9 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240612014219-fbbf4953d986 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.54.0 // indirect
	github.com/prometheus/procfs v0.15.0 // indirect
	github.com/richardartoul/molecule v1.0.1-0.20240531184615-7ca0df43c0b3 // indirect
	github.com/robfig/cron v1.2.0
	github.com/secure-systems-lab/go-securesystemslib v0.7.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.3 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/tinylib/msgp v1.2.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/oauth2 v0.22.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiextensions-apiserver v0.29.3 // indirect
	k8s.io/component-base v0.29.3 // indirect
	k8s.io/kube-openapi v0.0.0-20240228011516-70dd3763d340 // indirect
	k8s.io/utils v0.0.0-20240310230437-4693a0247e57 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)

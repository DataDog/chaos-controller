FROM ubuntu:23.10 as binaries

RUN apt-get update && \
    # iproute2 => tc
    # coreutils => df
    # iptables => iptables libs
    # libelf1 => EBPF libs
    apt-get -y install --no-install-recommends iproute2 coreutils iptables libelf1 tree curl tar ca-certificates && \
    # make copy from binaries unified and possible
    mkdir -p /lib64 && \
    curl -LO https://github.com/libbpf/bpftool/releases/download/v7.2.0/bpftool-v7.2.0-arm64.tar.gz && \
    echo "67a7eb48257594b7108442c262a8749fcd4dd2d15a38aab7d29cbb8745124822fa9039d3aff157261177d4ae99793edb77caca9332ab7bc5bc21ed295b208717  bpftool-v7.2.0-arm64.tar.gz" | sha512sum --check --status && \
    tar -xvf bpftool-v7.2.0-arm64.tar.gz && \
    chmod +x bpftool && \
    mv bpftool /usr/bin/bpftool-arm64 && \
    curl -LO https://github.com/libbpf/bpftool/releases/download/v7.2.0/bpftool-v7.2.0-amd64.tar.gz && \
    echo "1c991aff7d07010d6a668c1630cd8500aeecfffe9363823d019ede1eb8e8805b577499a67e5d8a80ce79c7f520a2c8a464fdf76c47a9c2690e57f54bf2fb8a82  bpftool-v7.2.0-amd64.tar.gz" | sha512sum --check --status && \
    tar -xvf bpftool-v7.2.0-amd64.tar.gz && \
    chmod +x bpftool && \
    mv bpftool /usr/bin/bpftool-amd64


FROM gcr.io/distroless/python3-debian11:latest

ARG TARGETARCH

# binaries used by the chaos-injector, ran as commmands
COPY --from=binaries /usr/bin/uname /usr/bin/df /usr/bin/ls /usr/bin/test /usr/bin/
COPY --from=binaries /usr/sbin/iptables /usr/sbin/
COPY --from=binaries /usr/sbin/tc /sbin/tc
COPY --from=binaries /usr/bin/bpftool-${TARGETARCH} /usr/bin/bpftool

# libraries used by above mentioned binaries (mostly GLIBC related)
COPY --from=binaries /lib/ld-linux-aarch64.so.[1]  /lib/
COPY --from=binaries /lib64/ld-linux-x86-64.so.[2] /lib64/
COPY --from=binaries /lib/aarch64-linux-gn[u] /lib/aarch64-linux-gnu/
COPY --from=binaries /lib/x86_64-linux-gn[u] /lib/x86_64-linux-gnu/
COPY --from=binaries /usr/lib/aarch64-linux-gn[u] /usr/lib/aarch64-linux-gnu/
COPY --from=binaries /usr/lib/x86_64-linux-gn[u] /usr/lib/x86_64-linux-gnu/

# no more sh
COPY --from=binaries /usr/bin/test /bin/sh

COPY injector_${TARGETARCH} /usr/local/bin/chaos-injector
COPY injector_${TARGETARCH} /usr/local/bin/injector

COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py
COPY ebpf/ /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/chaos-injector"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/python3-debian11:latest"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

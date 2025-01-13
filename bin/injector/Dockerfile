FROM ubuntu:24.04 AS binaries

RUN apt-get update && \
    # iproute2 => tc
    # coreutils => df
    # iptables => iptables libs
    # libelf1 => EBPF libs
    apt-get -y install --no-install-recommends iproute2 coreutils iptables libelf1 tree curl tar ca-certificates && \
    # make copy from binaries unified and possible
    mkdir -p /lib64 && \
    curl -LO https://github.com/libbpf/bpftool/releases/download/v7.5.0/bpftool-v7.5.0-arm64.tar.gz && \
    echo "7842789acac1c580b76c44437df791d1d2d07d5c282e1ef7034bd29a530ea8079478dfeb86ef7395ecae3127d3dcc3e9ebac4d5a77f421047a750b320bd25080  bpftool-v7.5.0-arm64.tar.gz" | sha512sum --check --status && \
    tar -xvf bpftool-v7.5.0-arm64.tar.gz && \
    chmod +x bpftool && \
    mv bpftool /usr/bin/bpftool-arm64 && \
    curl -LO https://github.com/libbpf/bpftool/releases/download/v7.5.0/bpftool-v7.5.0-amd64.tar.gz && \
    echo "742b05fff78d998b07a9c6c099ed6b7e8630072c98ae707c05fd8756327afc72cf7ed4afca5524a10544f28344020b056f54bd404f81e3c247b1ce4f8ba0e6c5  bpftool-v7.5.0-amd64.tar.gz" | sha512sum --check --status && \
    tar -xvf bpftool-v7.5.0-amd64.tar.gz && \
    chmod +x bpftool && \
    mv bpftool /usr/bin/bpftool-amd64


FROM gcr.io/distroless/python3-debian12:latest

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
COPY ebpf/${TARGETARCH}/ /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/chaos-injector"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/python3-debian12:latest"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

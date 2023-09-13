FROM ubuntu:23.10 as binaries

RUN apt-get update && \
    # iproute2 => tc
    # coreutils => df
    # iptables => iptables libs
    # libelf1 => EBPF libs
    apt-get -y install --no-install-recommends iproute2 coreutils iptables libelf1 tree && \
    # make copy from binaries unified and possible
    mkdir -p /lib64

FROM gcr.io/distroless/python3-debian11:latest

# binaries used by the chaos-injector, ran as commmands
COPY --from=binaries /usr/bin/df /usr/bin/ls /usr/bin/test /usr/bin/
COPY --from=binaries /usr/sbin/iptables /usr/sbin/
COPY --from=binaries /usr/sbin/tc /sbin/tc

# libraries used by above mentioned binaries (mostly GLIBC related)
COPY --from=binaries /lib/ld-linux-aarch64.so.[1]  /lib/
COPY --from=binaries /lib64/ld-linux-x86-64.so.[2] /lib64/
COPY --from=binaries /lib/aarch64-linux-gn[u] /lib/aarch64-linux-gnu/
COPY --from=binaries /lib/x86_64-linux-gn[u] /lib/x86_64-linux-gnu/
COPY --from=binaries /usr/lib/aarch64-linux-gn[u] /usr/lib/aarch64-linux-gnu/
COPY --from=binaries /usr/lib/x86_64-linux-gn[u] /usr/lib/x86_64-linux-gnu/

# no more sh
COPY --from=binaries /usr/bin/test /bin/sh

ARG TARGETARCH

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

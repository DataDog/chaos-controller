FROM ubuntu:22.10 as binaries

RUN apt-get update && \
    # iproute2 => tc
    # coreutils => df
    # iptables => iptables libs
    # libelf1 => EBPF libs
    apt-get -y install --no-install-recommends iproute2 coreutils iptables libelf1 tree

FROM gcr.io/distroless/python3-debian11:latest

# binaries used by the chaos-injector, ran as commmands
COPY --from=binaries /usr/bin/df /usr/bin/ls /usr/bin/test /usr/bin/
COPY --from=binaries /usr/sbin/iptables /usr/sbin/ip /usr/sbin/
COPY --from=binaries /sbin/tc /sbin/tc

# libraries used by above mentioned binaries (mostly GLIBC related)
ARG LIBARCH
COPY --from=binaries /lib/ld-linux-${LIBARCH}.so.1 /lib/
COPY --from=binaries /usr/lib/ld-linux-${LIBARCH}.so.1 /usr/lib/
COPY --from=binaries /lib/tc /lib/tc/
COPY --from=binaries /usr/lib/tc /usr/lib/tc/
COPY --from=binaries /lib/${LIBARCH}-linux-gnu /lib/${LIBARCH}-linux-gnu/
COPY --from=binaries /usr/lib/${LIBARCH}-linux-gnu /usr/lib/${LIBARCH}-linux-gnu/

# no more sh
COPY --from=binaries /usr/bin/test /bin/sh

ARG TARGETARCH
ENV BPF_DISK_FAILURE_NAME "bpf-disk-failure-${TARGETARCH}"

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

FROM ubuntu:22.10 as injector

ARG TARGETARCH
ENV BPF_DISK_FAILURE_NAME "bpf-disk-failure-${TARGETARCH}"

RUN apt-get update && \
    apt-get -y install curl git gcc iproute2 coreutils python3 iptables libelf1

COPY injector_${TARGETARCH} /usr/local/bin/injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py
COPY ebpf/ /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/injector"]

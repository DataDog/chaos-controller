FROM ubuntu:focal as injector

ARG TARGETARCH
ENV BPF_DISKFAILURE_NAME "bpf-diskfailure-${TARGETARCH}"

RUN apt-get update
RUN apt-get -y install curl git gcc iproute2 coreutils python3 iptables libelf1

COPY injector_${TARGETARCH} /usr/local/bin/injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py
COPY bpf-diskfailure-${TARGETARCH} /usr/local/bin/bpf-diskfailure
COPY bpf-diskfailure-${TARGETARCH}.bpf.o /bpf-diskfailure-${TARGETARCH}.bpf.o

RUN curl -LO http://ftp.de.debian.org/debian/pool/main/g/glibc/libc6_2.36-8_${TARGETARCH}.deb && \
    dpkg -i libc6_2.36-8_${TARGETARCH}.deb

ENTRYPOINT ["/usr/local/bin/injector"]

FROM ubuntu:focal as injector

ARG TARGETARCH
ENV BPF_DISKFAILURE_NAME "bpf-diskfailure-${TARGETARCH}"

RUN apt-get update
RUN apt-get -y install curl git gcc iproute2 coreutils python3 iptables libelf1

COPY injector_${TARGETARCH} /usr/local/bin/injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py

RUN curl -LO http://ftp.de.debian.org/debian/pool/main/g/glibc/libc6_2.36-8_${TARGETARCH}.deb && \
    dpkg -i libc6_2.36-8_${TARGETARCH}.deb

RUN curl -L  https://github.com/trailofbits/ebpfault/releases/download/v1.1.0/ebpfault-1.1.0-1.x86_64.tar.gz > ebpfault.tar.gz

RUN curl -LO  https://github.com/aymericDD/bpf-diskfailure/releases/download/v0.1.0/${BPF_DISKFAILURE_NAME}.tar.gz && \
    mkdir -p ${BPF_DISKFAILURE_NAME}

RUN export BPF_DISKFAILURE_NAME_ARCH="$(echo $BPF_DISKFAILURE_NAME | sed -e 's/amd64/x86/')" && \
    tar xvf ${BPF_DISKFAILURE_NAME}.tar.gz -C ${BPF_DISKFAILURE_NAME} && \
    mv ${BPF_DISKFAILURE_NAME}/$BPF_DISKFAILURE_NAME_ARCH.bpf.o /$BPF_DISKFAILURE_NAME_ARCH.bpf.o && \
    mv ${BPF_DISKFAILURE_NAME}/${BPF_DISKFAILURE_NAME_ARCH} /usr/local/bin/bpf-diskfailure && \
    rm -rf ${BPF_DISKFAILURE_NAME}/ ${BPF_DISKFAILURE_NAME}.tar.gz

ENTRYPOINT ["/usr/local/bin/injector"]

FROM ubuntu:focal as injector

ARG TARGETARCH

RUN apt-get update
RUN apt-get -y install curl git gcc iproute2 coreutils python3 iptables

COPY injector_${TARGETARCH} /usr/local/bin/injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py
RUN curl -L  https://github.com/trailofbits/ebpfault/releases/download/v1.1.0/ebpfault-1.1.0-1.x86_64.tar.gz > ebpfault.tar.gz && \
    tar xvfz ebpfault.tar.gz && \
    mv ebpfault-1.1.0-1.x86_64/bin/ebpfault /usr/bin/ && \
    rm -rf ebpfault-1.1.0-1.x86_64 ebpfault.tar.gz

ENTRYPOINT ["/usr/local/bin/injector"]

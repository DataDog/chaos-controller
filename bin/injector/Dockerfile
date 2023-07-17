FROM ubuntu:22.10 as injector

ARG TARGETARCH

RUN apt-get update && \
    apt-get -y install curl git gcc iproute2 coreutils python3 iptables libelf1

RUN apt-get update && \
    apt-get install -y \
    curl \
    pahole \
    dwarves \
    make \
    clang \
    libelf-dev \
    libbpf-dev \
    llvm \
    strace

RUN rm /usr/sbin/bpftool; \
    cd / && git clone --recurse-submodules https://github.com/libbpf/bpftool.git && \
    cd bpftool/src && \
    make install && \
    ln -s /usr/local/sbin/bpftool /usr/sbin/bpftool

COPY injector_${TARGETARCH} /usr/local/bin/chaos-injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py
COPY ebpf/ /usr/local/bin/

# create a symlink to not break if anyone used explicitly injector somewhere
RUN ln -s /usr/local/bin/chaos-injector /usr/local/bin/injector

ENTRYPOINT ["/usr/local/bin/chaos-injector"]

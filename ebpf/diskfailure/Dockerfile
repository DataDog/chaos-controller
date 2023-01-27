FROM ubuntu:22.10

# Define variables.
ARG GOVERSION=1.19.4
ARG ARCH=arm64

# Download development environment.
RUN apt-get update || apt-get update && \
    apt-get install -y \
        curl \
        pahole \
        dwarves \
        make \
        clang \
        libelf-dev \
        libbpf-dev \
        llvm

RUN if [ "$ARCH" = "amd64" ] ; then apt-get install -y libc6-dev-i386 ; fi

# Install Go specific version.
RUN apt-get install -y wget && \
    curl -LO https://golang.org/dl/go${GOVERSION}.linux-${ARCH}.tar.gz && \
    tar -xf go${GOVERSION}.linux-${ARCH}.tar.gz && \
    mv go/ /usr/local/ && \
    ln -s /usr/local/go/bin/go /usr/local/bin/ && \
    rm -f go${GOVERSION}.linux-${ARCH}.tar.gz

# Install libpf from source
#RUN export UNAME=$(uname -m) && \
#    curl -LO https://github.com/libbpf/libbpf/archive/refs/tags/v1.1.0.tar.gz  && \
#    tar xvf v1.1.0.tar.gz  && \
#    cd libbpf-1.1.0/src && \
#    NO_PKG_CONFIG=1  DESTDIR=/build/root make install && \
#    cp -r /build/root/usr/include/bpf/ /usr/include/bpf && \
#    cp /build/root/usr/lib64/libbpf.a "/usr/lib/$UNAME-linux-gnu/libbpf.a"


# Setup working directory.
RUN mkdir -p /app
WORKDIR /app

# Execute build command.
ENTRYPOINT ["/usr/bin/make", "all"]

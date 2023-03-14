FROM ubuntu:22.10

# Define variables.
ARG GOVERSION=1.19.4
ARG ARCH=arm64

# Download development environment.
RUN apt-get update && \
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

# Setup working directory.
RUN mkdir -p /app
WORKDIR /app/ebpf

# Execute build command.
ENTRYPOINT ["/usr/bin/make", "all"]

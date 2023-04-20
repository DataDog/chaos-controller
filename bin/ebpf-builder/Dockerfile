FROM ubuntu:22.10

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

ARG ARCH=arm64
RUN ([ "${ARCH}" = "amd64" ] && apt-get install -y libc6-dev-i386) || true

# Install Go specific version.
ENV GOVERSION=1.19.4
RUN curl -LO https://golang.org/dl/go${GOVERSION}.linux-${ARCH}.tar.gz && \
    tar -xf go${GOVERSION}.linux-${ARCH}.tar.gz && \
    mv go/ /usr/local/ && \
    ln -s /usr/local/go/bin/go /usr/local/bin/ && \
    rm -f go${GOVERSION}.linux-${ARCH}.tar.gz

# Setup working directory.
RUN mkdir -p /app
WORKDIR /app/ebpf

# Execute build command.
ENTRYPOINT ["/usr/bin/make", "all"]

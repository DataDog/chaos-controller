FROM ubuntu:23.10

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

ARG TARGETARCH
RUN ([ "${TARGETARCH}" = "amd64" ] && apt-get install -y libc6-dev-i386) || true

# Provide expected go version through build arg
ARG BUILDGOVERSION
# We defined TARGETARCH and BUILDGOVERSION as env so they can be seen by the make install-go target
ENV TARGETARCH=${TARGETARCH} \
    BUILDGOVERSION=${BUILDGOVERSION}
WORKDIR /tmp
COPY ./scripts/install-go /tmp/install-go
RUN /tmp/install-go && rm -f /tmp/install-go

# Setup working directory.
WORKDIR /app/ebpf

# Execute build command.
ENTRYPOINT ["/usr/bin/make", "all"]

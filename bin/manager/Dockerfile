# syntax=docker/dockerfile:1
# Build stage
ARG BUILDGOVERSION=1.27.3
FROM --platform=$BUILDPLATFORM golang:${BUILDGOVERSION} AS builder

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /go/src/github.com/chaos-controller/

# Copy vendored dependencies first for better layer caching (avoids network downloads)
COPY go.mod go.sum ./
COPY vendor/ vendor/

# Copy the source code
COPY . .

# Build the manager binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -mod=vendor -ldflags="-s -w" -o manager .
# Prebuilt binary stage (used by CI when the Go binary is compiled on the host)
FROM gcr.io/distroless/base-debian13:nonroot AS prebuilt

COPY --chmod=100 bin/manager/manager /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/base-debian13:nonroot"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

# Final stage
FROM gcr.io/distroless/base-debian13:nonroot

COPY --chmod=100 --from=builder /go/src/github.com/chaos-controller/manager /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/base-debian13:nonroot"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

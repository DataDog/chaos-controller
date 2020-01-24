# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace

COPY go.mod .
COPY go.sum .

# Cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy sources
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -ldflags="-w -s" -a -o bin/manager main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -ldflags="-w -s" -a -o bin/injector ./cli/injector

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot as manager
WORKDIR /
COPY --from=builder /workspace/bin/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]

# Injector image
FROM alpine:3.11.2 as injector
RUN apk update && \
    apk add git gcc musl-dev iptables iproute2
WORKDIR /
COPY --from=builder /workspace/bin/injector .
ENTRYPOINT ["/injector"]

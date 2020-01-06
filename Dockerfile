# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/github.com/DataDog/chaos-fi-controller
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o manager github.com/DataDog/chaos-fi-controller/cmd/manager
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -a -o injector github.com/DataDog/chaos-fi-controller/cmd/injector

# Manager image
FROM scratch as manager
WORKDIR /
COPY --from=builder /go/src/github.com/DataDog/chaos-fi-controller/manager .
ENTRYPOINT ["/manager"]

# Injector image
FROM alpine:3.11.2 as injector
RUN apk update && \
    apk add git gcc musl-dev iptables
WORKDIR /
COPY --from=builder /go/src/github.com/DataDog/chaos-fi-controller/injector .
ENTRYPOINT ["/injector"]

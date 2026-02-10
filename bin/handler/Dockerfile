# Build stage
ARG BUILDGOVERSION=1.25.6
FROM golang:${BUILDGOVERSION} AS builder

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /go/src/github.com/chaos-controller/

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the handler binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o handler ./cli/handler

# Final stage
FROM scratch AS handler

COPY --chmod=100 --from=builder /go/src/github.com/chaos-controller/handler /usr/local/bin/handler

ENTRYPOINT ["/usr/local/bin/handler"]

LABEL baseimage.os="scratch"
LABEL baseimage.isgbi="scratch"
LABEL baseimage.name="scratch"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

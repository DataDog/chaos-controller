# Build stage
FROM golang:1.25.3 AS builder

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /go/src/github.com/chaos-controller/

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the manager binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o manager .

# Final stage
FROM gcr.io/distroless/base-debian12:nonroot

COPY --chmod=100 --from=builder /go/src/github.com/chaos-controller/manager /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/base-debian12:nonroot"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

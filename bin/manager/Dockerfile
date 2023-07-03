FROM gcr.io/distroless/base-debian11:nonroot

ARG TARGETARCH
COPY manager_${TARGETARCH} /usr/local/bin/manager

# COPY go runtime as it's needed by ddmark to dynamically load structs and their associated markers
COPY --from=golang:1.20.5 /usr/local/go /usr/local/go/
COPY --from=golang:1.20.5 /usr/local/go/bin /usr/local/bin/

ENTRYPOINT ["/usr/local/bin/manager"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/base-debian11:nonroot"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

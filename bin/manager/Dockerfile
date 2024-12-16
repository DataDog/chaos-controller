FROM gcr.io/distroless/base-debian12:nonroot

ARG TARGETARCH
COPY manager_${TARGETARCH} /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

LABEL baseimage.os="debian"
LABEL baseimage.isgbi="custom"
LABEL baseimage.name="gcr.io/distroless/base-debian12:nonroot"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

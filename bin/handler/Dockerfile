FROM scratch as handler

ARG TARGETARCH
COPY handler_${TARGETARCH} /usr/local/bin/handler

ENTRYPOINT ["/usr/local/bin/handler"]

LABEL baseimage.os="scratch"
LABEL baseimage.isgbi="scratch"
LABEL baseimage.name="scratch"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

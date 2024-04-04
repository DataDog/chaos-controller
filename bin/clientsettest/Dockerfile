FROM scratch as clientsettest

ARG TARGETARCH
COPY clientsettest_${TARGETARCH} /usr/local/bin/clientsettest

ENTRYPOINT ["/usr/local/bin/clientsettest"]

LABEL baseimage.os="scratch"
LABEL baseimage.isgbi="scratch"
LABEL baseimage.name="scratch"

ARG BUILDSTAMP
LABEL baseimage.buildstamp="${BUILDSTAMP}"

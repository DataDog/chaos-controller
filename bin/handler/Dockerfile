FROM scratch as handler

ARG TARGETARCH

COPY handler_${TARGETARCH} /usr/local/bin/handler

ENTRYPOINT ["/usr/local/bin/handler"]

FROM gcr.io/distroless/static:nonroot as manager

USER nonroot:nonroot
ARG TARGETARCH

COPY manager_${TARGETARCH} /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

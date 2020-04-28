FROM gcr.io/distroless/static:nonroot as manager

USER nonroot:nonroot

COPY manager /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

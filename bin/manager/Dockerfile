FROM golang:1.18-alpine

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

ARG TARGETARCH
COPY manager_${TARGETARCH} /usr/local/bin/manager

ENTRYPOINT ["/usr/local/bin/manager"]

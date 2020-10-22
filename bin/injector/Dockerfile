FROM alpine:3.11.2 as injector

RUN apk update && \
    apk add git gcc musl-dev iproute2 coreutils

COPY injector /usr/local/bin/injector

ENTRYPOINT ["/usr/local/bin/injector"]

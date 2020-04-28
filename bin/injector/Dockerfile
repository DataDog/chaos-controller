FROM alpine:3.11.2 as injector

RUN apk update && \
    apk add git gcc musl-dev iptables iproute2

COPY injector /usr/local/bin/injector

ENTRYPOINT ["/usr/local/bin/injector"]

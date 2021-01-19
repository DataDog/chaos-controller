FROM alpine:3.11.2 as injector

RUN apk update && \
    apk add git gcc musl-dev iproute2 coreutils python3

COPY injector /usr/local/bin/injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py

ENTRYPOINT ["/usr/local/bin/injector"]

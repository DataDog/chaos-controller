FROM ubuntu:focal as injector

ARG TARGETARCH

RUN apt-get update && apt-get -y install git gcc iproute2 coreutils python3 iptables

COPY injector_${TARGETARCH} /usr/local/bin/injector
COPY dns_disruption_resolver.py /usr/local/bin/dns_disruption_resolver.py


ENTRYPOINT ["/usr/local/bin/injector"]

FROM golang:1.12.1-alpine

RUN apk update && \
    apk add git gcc musl-dev iptables make

WORKDIR /go/src/github.com/DataDog/chaos-fi
COPY . .

RUN make install

CMD ["chaos-fi"]

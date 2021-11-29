# ---------------------------------------
FROM amd64/golang:1.16-alpine as build

ENV GOARCH=amd64 
ENV CGO_ENABLED=0
WORKDIR /app

COPY go.* .
RUN go mod download
RUN go get github.com/markbates/pkger/cmd/pkger

COPY . .
RUN pkger -o ./cli/chaosli/cmd

RUN go build \
    -ldflags="-X github.com/DataDog/chaos-controller/cli/chaosli/cmd.Version=dockertest" \
    -o ./bin/chaosli/chaosli_bin \
    ./cli/chaosli

# ---------------------------------------
FROM amd64/golang:1.16-alpine as bin

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

COPY --from=build /app/bin/chaosli/chaosli_bin ./chaosli_bin
COPY --from=build /app/examples ./examples

RUN ./chaosli_bin validate --path ./examples/complete.yaml 

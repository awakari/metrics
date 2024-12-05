FROM golang:1.23.4-alpine3.20 AS builder
WORKDIR /go/src/metrics
COPY . .
RUN \
    apk add -U --no-cache \
        protoc \
        protobuf-dev \
        make \
        git \
        ca-certificates && \
    make build

FROM scratch
COPY --from=builder /go/src/metrics/metrics /bin/metrics
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/bin/metrics"]

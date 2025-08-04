FROM golang:1.24.5-alpine3.22 AS builder
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

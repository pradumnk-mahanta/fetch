ARG VERSION=dev
FROM golang:1.25-bookworm AS builder

ARG VERSION
WORKDIR /app

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-X fetch/config.version=$VERSION" -o fetch

FROM debian:bookworm-slim

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/fetch /app/fetch

RUN mkdir -p /downloads /data

VOLUME ["/downloads", "/data"]

ENTRYPOINT ["/app/fetch"]
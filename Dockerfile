FROM golang:1.22-bookworm AS builder

WORKDIR /app

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o fetch

FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/fetch /app/fetch

RUN mkdir -p /downloads /data

VOLUME ["/downloads", "/data"]

ENV APPLICATION_API_PORT=9090
ENV APPLICATION_LOG_LEVEL=INFO
ENV APPLICATION_DOWNLOAD_ROOT=/downloads
ENV APPLICATION_USENET_DOWNLOAD_PROVIDER=
ENV APPLICATION_MAX_DOWNLOAD_SEND_TO_PROVIDER=2
ENV SABNZBD_API_KEY=
ENV DOWNLOADER_MAX_PARALLEL_DOWNLOADS=2
ENV DOWNLOADER_MAX_RETRY_DOWNLOADS=2
ENV PROVIDER_TB_CONFIG_API_KEY=
ENV PROVIDER_TB_CONFIG_PREFER_ZIPPED_FOLDER=false

ENTRYPOINT ["/app/fetch"]
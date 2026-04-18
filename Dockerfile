FROM --platform=$BUILDPLATFORM tonistiigi/xx AS xx

ARG VERSION=dev

# -------------------------
# Node / Tailwind Builder
# -------------------------
FROM node:20-bookworm AS node-builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm install

COPY tailwind.config.js ./
COPY tailwind.css ./
COPY handlers ./handlers

RUN npx tailwindcss -i ./tailwind.css -o ./static/styles.css --minify


# -------------------------
# Go Builder
# -------------------------
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

COPY --from=xx / /

ARG VERSION
ARG TARGETPLATFORM

WORKDIR /app

RUN apt-get update && xx-apt-get install -y gcc g++ libc6-dev

ENV CGO_ENABLED=1

COPY go.mod go.sum ./
RUN go mod download

COPY . .

COPY --from=node-builder /app/static /app/static

RUN xx-go build -ldflags="-X fetch/config.version=$VERSION -s -w" -o fetch && xx-verify fetch


# -------------------------
# Runtime Image
# -------------------------
FROM debian:bookworm-slim

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates tzdata sqlite3 curl && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/fetch /app/fetch
COPY --from=builder /app/static /app/static

RUN mkdir -p /downloads /data

VOLUME ["/downloads", "/data"]

ENTRYPOINT ["/app/fetch"]
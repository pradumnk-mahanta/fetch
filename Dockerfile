# syntax=docker/dockerfile:1.6

ARG VERSION=dev

FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

ARG VERSION
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

ENV CGO_ENABLED=1
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH

# Install cross compilers for CGO
RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    gcc-x86-64-linux-gnu \
    libc6-dev-arm64-cross \
    libc6-dev-amd64-cross \
    && rm -rf /var/lib/apt/lists/*

# Select correct compiler based on target
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
    else \
        export CC=x86_64-linux-gnu-gcc; \
    fi && \
    echo "Using CC=$CC"

# Persist CC for later RUN steps
ENV CC=aarch64-linux-gnu-gcc

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Override CC dynamically again during build
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc; \
    else \
        export CC=x86_64-linux-gnu-gcc; \
    fi && \
    go build -ldflags="-X fetch/config.version=$VERSION" -o fetch

FROM --platform=$TARGETPLATFORM debian:bookworm-slim

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates tzdata sqlite3 && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/fetch /app/fetch
COPY --from=builder /app/logo.png /app/logo.png

RUN mkdir -p /downloads /data

VOLUME ["/downloads", "/data"]

ENTRYPOINT ["/app/fetch"]
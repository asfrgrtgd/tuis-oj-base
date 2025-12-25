# builder stage
FROM golang:1.23.2-alpine3.20@sha256:9dd2625a1ff2859b8d8b01d8f7822c0f528942fe56cfe7a1e7c38d3b8d72d679 AS builder
WORKDIR /app
ENV GOTOOLCHAIN=local
RUN apk add --no-cache curl
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o worker ./cmd/worker

# download migrate binary
ARG MIGRATE_VERSION=4.17.1
RUN curl -fsSL https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz \
    | tar -xz -C /tmp -f -

# runtime stage
FROM debian:12-slim@sha256:e899040a73d36e2b36fa33216943539d9957cba8172b858097c2cabcdb20a3e2
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 65532 appuser \
    && mkdir -p /var/log/oj \
    && chown -R appuser:appuser /var/log/oj

COPY --from=builder /app/server /app/server
COPY --from=builder /app/worker /app/worker
COPY --from=builder /tmp/migrate /usr/local/bin/migrate

ENV PORT=3000
ENV LOG_DIR=/var/log/oj
EXPOSE 3000
USER appuser
ENTRYPOINT ["/app/server"]

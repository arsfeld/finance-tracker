# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY src/ ./src/

ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o finance-tracker ./src

# Final stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/finance-tracker .

# LinuxServer.io convention: all persistent data under /config
RUN mkdir -p /config
VOLUME /config

ENV DATA_DIR=/config

CMD ["./finance-tracker", "--env-file", "/config/.env"]

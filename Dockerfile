# Stage 1: Build React frontend
FROM node:22-alpine AS frontend-build
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/

ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" \
    -o finance-server ./cmd/server

# Stage 3: Final image
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=backend-build /app/finance-server .
COPY --from=frontend-build /app/web/dist ./web/dist/

RUN mkdir -p /config
VOLUME /config

ENV DATA_DIR=/config
ENV DB_PATH=/config/finance_tracker.db
ENV ENV_FILE=/config/.env
ENV FRONTEND_DIR=/app/web/dist
ENV LISTEN_ADDR=:8080

EXPOSE 8080

CMD ["./finance-server"]

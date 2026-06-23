# Stage 1: Build React frontend.
# The frontend output is static JS, so always build it on the native build
# platform regardless of the target arch (avoids emulating npm under QEMU).
FROM --platform=$BUILDPLATFORM node:22-alpine AS frontend-build
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend.
# Build on the native platform and cross-compile to the target arch via the
# buildx-provided TARGETOS/TARGETARCH args. CGO is disabled, so this is a fast
# pure-Go cross-compile with no QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS backend-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/

ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
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

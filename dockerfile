FROM rust:1.82-slim AS builder

RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    libpq-dev \
    libsqlite3-dev \
    libz-dev \
    libffi-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /usr/src/

COPY . .

RUN cargo build --release

FROM node:20 AS ui-builder

RUN npm install -g pnpm@9.14.4

WORKDIR /usr/src/ui

COPY ui/package.json .
COPY ui/pnpm-lock.yaml .

RUN pnpm install --frozen-lockfile

COPY ui/ .

RUN pnpm build

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    openssl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

ENV STORAGE_PATH=/app/data

COPY --from=builder /usr/src/config /app/config
COPY --from=builder /usr/src/assets /app/assets
COPY --from=builder /usr/src/target/release/finance_tracker-cli /app/finance_tracker-cli
COPY --from=ui-builder /usr/src/ui/dist /app/ui/dist

VOLUME /app/data

ENTRYPOINT ["/app/finance_tracker-cli"]
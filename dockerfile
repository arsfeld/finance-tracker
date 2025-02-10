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

WORKDIR /usr/app

COPY --from=builder /usr/src/config /usr/app/config
COPY --from=builder /usr/src/assets /usr/app/assets
COPY --from=builder /usr/src/target/release/finance_tracker-cli /usr/app/finance_tracker-cli
COPY --from=ui-builder /usr/src/ui/dist /usr/app/ui/dist

ENTRYPOINT ["/usr/app/finance_tracker-cli"]
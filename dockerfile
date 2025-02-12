FROM lukemathwalker/cargo-chef:latest-rust-1 AS chef

WORKDIR /app

FROM chef AS planner

COPY . .

RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder 

RUN apt-get update && apt-get install -y \
    pkg-config \
    libssl-dev \
    libpq-dev \
    libsqlite3-dev \
    libz-dev \
    libffi-dev \
    && rm -rf /var/lib/apt/lists/*

COPY --from=planner /app/recipe.json recipe.json

# Build dependencies - this is the caching Docker layer!
RUN cargo chef cook --release --recipe-path recipe.json

# Build application
COPY . .

RUN cargo build --release

FROM node:20 AS ui-builder

RUN npm install -g pnpm@9.14.4

WORKDIR /app/ui

COPY ui/package.json .
COPY ui/pnpm-lock.yaml .

RUN pnpm install --frozen-lockfile

COPY ui/ .

RUN pnpm build

FROM debian:bookworm-slim AS runner

RUN apt-get update && apt-get install -y \
    openssl ca-certificates supervisor\
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

ENV STORAGE_PATH=/app/data
ENV LOCO_ENV=production

COPY --from=builder /app/config /app/config
COPY --from=builder /app/assets /app/assets
COPY --from=builder /app/target/release/finance_tracker-cli /app/finance_tracker-cli
COPY --from=ui-builder /app/ui/dist /app/ui/dist

COPY supervisor.conf /etc/supervisor/conf.d/supervisor.conf

VOLUME /app/data

CMD ["supervisord", "-c", "/etc/supervisor/conf.d/supervisor.conf"]
FROM lukemathwalker/cargo-chef:latest-rust-1 AS chef

WORKDIR /app

FROM chef AS planner

COPY . .

RUN cargo chef prepare --recipe-path recipe.json

FROM chef AS builder 

COPY --from=planner /app/recipe.json recipe.json

# Build dependencies - this is the caching Docker layer!
RUN cargo chef cook --release --recipe-path recipe.json

# Build application
COPY . .

RUN cargo build --release

FROM debian:bookworm-slim AS runner

WORKDIR /app

COPY --from=builder /app/target/release/finance-tracker /app/finance-tracker
COPY --from=builder /app/templates /app/templates

CMD ["/app/finance-tracker"]

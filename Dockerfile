# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./src/

# Build the application with version information
ARG VERSION=dev
ARG BUILD_TIME
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o finance-tracker ./src

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/finance-tracker .

# Expose port if needed (uncomment if your app needs it)
# EXPOSE 8080

# Run the application
CMD ["./finance-tracker"]

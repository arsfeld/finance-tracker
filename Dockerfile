# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o finance-tracker

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/finance-tracker .
# Copy templates directory
COPY --from=builder /app/templates ./templates

# Expose port if needed (uncomment if your app needs it)
# EXPOSE 8080

# Run the application
CMD ["./finance-tracker"]

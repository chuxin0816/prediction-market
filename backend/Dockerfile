# Build stage
FROM golang:1.23 AS builder

WORKDIR /app

# Enable automatic toolchain download
ENV GOTOOLCHAIN=auto

# Copy all source files first
COPY . .

# Download dependencies and build in a single layer so toolchain persists
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/server .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./server"]

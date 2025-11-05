# Multi-stage Dockerfile for ReqTap

# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install necessary tools
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o reqtap ./cmd/reqtap

# Runtime stage
FROM alpine:latest

# Install ca-certificates to support HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S reqtap && \
    adduser -u 1001 -S reqtap -G reqtap

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/reqtap .
COPY --from=builder /app/configs ./configs

# Create log directory
RUN mkdir -p /app/logs && \
    chown -R reqtap:reqtap /app

# Switch to non-root user
USER reqtap

# Expose port
EXPOSE 38888

# Set default environment variables
ENV REQTAP_SERVER_PORT=38888
ENV REQTAP_SERVER_PATH="/"
ENV REQTAP_LOG_LEVEL="info"

# Health check configuration
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:38888/ || exit 1

# Startup command
CMD ["./reqtap"]
# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o gohttp cmd/server/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /app

# Copy the binary and config from builder stage
COPY --from=builder /app/gohttp .
COPY --from=builder /app/config.json .
COPY --from=builder /app/static ./static

# Create non-root user
RUN addgroup -g 1001 -S gohttp && \
    adduser -S gohttp -u 1001 -G gohttp

# Change ownership of app directory
RUN chown -R gohttp:gohttp /app

# Switch to non-root user
USER gohttp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/stats || exit 1

# Run the application
CMD ["./gohttp"]
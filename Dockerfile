# Build stage
FROM golang:1.23-alpine AS builder

# Set build arguments
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG GIT_BRANCH

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags "-s -w \
    -X 'frizo/futures_engine/internal/version.Version=${VERSION}' \
    -X 'frizo/futures_engine/internal/version.BuildTime=${BUILD_TIME}' \
    -X 'frizo/futures_engine/internal/version.GitCommit=${GIT_COMMIT}' \
    -X 'frizo/futures_engine/internal/version.GitBranch=${GIT_BRANCH}'" \
    -o futures_engine ./cmd/futures_engine

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/futures_engine .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Change ownership to appuser
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port (adjust as needed)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./futures_engine --health-check || exit 1

# Run the binary
ENTRYPOINT ["./futures_engine"]
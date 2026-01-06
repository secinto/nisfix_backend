# =============================================================================
# NisFix Backend Dockerfile
# Multi-stage build for minimal production image
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Build arguments for versioning
ARG VERSION=0.1.0-dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown
ARG GIT_BRANCH=unknown

# Set working directory
WORKDIR /build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "\
        -X main.Version=${VERSION} \
        -X main.BuildTime=${BUILD_TIME} \
        -X main.GitCommit=${GIT_COMMIT} \
        -X main.GitBranch=${GIT_BRANCH} \
        -s -w" \
    -o /build/nisfix-backend \
    ./cmd/server

# -----------------------------------------------------------------------------
# Stage 2: Runtime
# -----------------------------------------------------------------------------
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user for security
RUN addgroup -S nisfix && adduser -S nisfix -G nisfix

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/nisfix-backend /app/nisfix-backend

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Create directories for keys and data
RUN mkdir -p /app/keys /app/data && \
    chown -R nisfix:nisfix /app

# Switch to non-root user
USER nisfix

# Expose the application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

# Set default environment
ENV NISFIX_SERVER_PORT=8080
ENV NISFIX_ENVIRONMENT=production

# Run the application
ENTRYPOINT ["/app/nisfix-backend"]

# Frontend build stage (optional - only runs if frontend source exists)
# Use the builder's native platform so npm/vite run without QEMU emulation.
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend files
COPY frontend/package*.json ./
COPY frontend/ ./

# Install dependencies and build (if package.json exists)
# Note: Removed --production flag because devDependencies (typescript, vite) are needed for build
RUN if [ -f "package.json" ]; then \
      npm ci && \
      npm run build; \
    fi

# Backend build stage
# Use the builder's native platform so the Go toolchain runs without QEMU emulation.
# CGO_ENABLED=0 with GOOS/GOARCH lets Go cross-compile natively to linux/amd64.
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS backend-builder

# Install security updates and build dependencies
RUN apk update && apk upgrade && \
    apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies (cached across builds so unchanged modules aren't re-fetched)
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source code
COPY . .

# Copy frontend dist so //go:embed can include it at compile time
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Build the application. The Go build cache mount persists compiled packages
# across builds, so only changed packages are recompiled instead of the
# whole dependency tree (e.g. Azure SDK) every time.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s -extldflags "-static"' -o /app/api ./cmd/api

# Final stage — use the pre-built base image so LibreOffice is not installed
# on every build. Rebuild the base by running build-base-image.yml manually
# or by editing Dockerfile.base (it also rebuilds monthly for security patches).
# Tag format: YYYY.MM — update this after each monthly base rebuild
# (see build-base-image.yml; configure Renovate to automate the bump).
FROM ghcr.io/networkengineer-cloud/go-volunteer-media-base:2026.06

# Copy binary from backend builder
COPY --from=backend-builder /app/api /api

# Copy public directory for uploads and static assets
COPY --from=backend-builder /app/public /public

# Create uploads directory and set permissions
RUN mkdir -p /public/uploads && \
    chown -R appuser:appuser /public/uploads && \
    chmod -R 755 /public/uploads

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/api"]

# Frontend build stage (optional - only runs if frontend source exists)
# Use the builder's native platform so npm/vite run without QEMU emulation.
FROM --platform=$BUILDPLATFORM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Vite inlines VITE_-prefixed env vars into import.meta.env.* at build time
# (not read at container runtime), and .env files are excluded from the
# build context by .dockerignore - so this must arrive as a build-arg from
# CI (build-image.yml) rather than a .env file. Not a secret: LaunchDarkly's
# client-side ID is designed to be exposed in browser JS.
ARG VITE_LAUNCHDARKLY_CLIENT_ID=""
ENV VITE_LAUNCHDARKLY_CLIENT_ID=${VITE_LAUNCHDARKLY_CLIENT_ID}

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
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS backend-builder

# GIT_SHA identifies the build for the service.version OTel resource
# attribute (see internal/version). Passed in via --build-arg from CI
# (build-image.yml); defaults to "dev" for a manual `docker build` with no
# build-arg, matching internal/version.GitSHA's own local-dev default.
ARG GIT_SHA=dev

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
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -extldflags \"-static\" -X github.com/networkengineer-cloud/go-volunteer-media/internal/version.GitSHA=${GIT_SHA}" -o /app/api ./cmd/api

# Final stage — use the pre-built base image so LibreOffice is not installed
# on every build. Rebuild the base by running build-base-image.yml manually
# or by editing Dockerfile.base (it also rebuilds monthly for security patches).
# Tag format: YYYY.MM — Renovate (see renovate.json) opens a PR bumping this
# after each monthly rebuild in build-base-image.yml.
FROM ghcr.io/networkengineer-cloud/go-volunteer-media-base:2026.09

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

# Frontend build stage (optional - only runs if frontend source exists)
FROM node:20-alpine AS frontend-builder

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
FROM golang:1.24-alpine AS backend-builder

# Install security updates and build dependencies
RUN apk update && apk upgrade && \
    apk add --no-cache git ca-certificates tzdata && \
    update-ca-certificates

# Create non-root user
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s -extldflags "-static"' -o /app/api ./cmd/api

# Final stage
FROM scratch

# Copy certificates, timezone data, and user info from builder
COPY --from=backend-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=backend-builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=backend-builder /etc/passwd /etc/passwd

# Copy binary from backend builder
COPY --from=backend-builder /app/api /api

# Copy frontend build if it exists
COPY --from=frontend-builder /app/frontend/dist /frontend/dist

# Copy public directory for uploads and static assets
COPY --from=backend-builder /app/public /public

# Use non-root user
USER appuser

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/api"]

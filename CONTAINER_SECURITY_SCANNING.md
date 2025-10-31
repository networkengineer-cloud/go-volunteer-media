# Container Security Scanning Guide

This document provides guidance on implementing container image security scanning in CI/CD pipelines for the Haws Volunteer Media application.

## Overview

Container security scanning helps identify vulnerabilities in Docker images before they reach production. This guide covers integration with popular CI/CD platforms and scanning tools.

## Scanning Tools

### 1. Trivy (Recommended)

Trivy is a comprehensive vulnerability scanner for containers and other artifacts.

#### Installation
```bash
# Linux
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
sudo apt-get update
sudo apt-get install trivy

# macOS
brew install trivy

# Docker
docker pull aquasec/trivy:latest
```

#### Usage
```bash
# Scan local image
trivy image volunteer-media-api:latest

# Scan with specific severity
trivy image --severity HIGH,CRITICAL volunteer-media-api:latest

# Generate JSON report
trivy image --format json --output report.json volunteer-media-api:latest

# Scan Dockerfile
trivy config Dockerfile

# Fail build on high/critical vulnerabilities
trivy image --exit-code 1 --severity HIGH,CRITICAL volunteer-media-api:latest
```

### 2. Snyk Container

Snyk provides developer-friendly vulnerability scanning.

#### Installation
```bash
npm install -g snyk

# Authenticate
snyk auth
```

#### Usage
```bash
# Scan image
snyk container test volunteer-media-api:latest

# Monitor image for new vulnerabilities
snyk container monitor volunteer-media-api:latest

# Generate HTML report
snyk container test volunteer-media-api:latest --json | snyk-to-html -o report.html
```

### 3. Grype

Grype is a vulnerability scanner for container images and filesystems.

#### Installation
```bash
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
```

#### Usage
```bash
# Scan image
grype volunteer-media-api:latest

# Fail on high/critical
grype volunteer-media-api:latest --fail-on high

# Output formats
grype volunteer-media-api:latest -o json
grype volunteer-media-api:latest -o sarif
```

### 4. Docker Scout (Built into Docker)

Docker Scout is integrated into Docker Desktop and CLI.

#### Usage
```bash
# Enable Docker Scout
docker scout quickview

# Scan image
docker scout cves volunteer-media-api:latest

# Compare images
docker scout compare --to volunteer-media-api:old volunteer-media-api:latest
```

## CI/CD Integration Examples

### GitHub Actions

Create `.github/workflows/security-scan.yml`:

```yaml
name: Container Security Scan

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]
  schedule:
    # Run daily at 2 AM UTC
    - cron: '0 2 * * *'

jobs:
  scan-trivy:
    name: Trivy Security Scan
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Build Docker image
        run: docker build -t volunteer-media-api:${{ github.sha }} .

      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: volunteer-media-api:${{ github.sha }}
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'

      - name: Upload Trivy results to GitHub Security tab
        uses: github/codeql-action/upload-sarif@v2
        if: always()
        with:
          sarif_file: 'trivy-results.sarif'

      - name: Run Trivy with table output
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: volunteer-media-api:${{ github.sha }}
          format: 'table'
          exit-code: '1'
          ignore-unfixed: true
          severity: 'CRITICAL,HIGH'

  scan-snyk:
    name: Snyk Security Scan
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Build Docker image
        run: docker build -t volunteer-media-api:${{ github.sha }} .

      - name: Run Snyk to check Docker image
        uses: snyk/actions/docker@master
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
        with:
          image: volunteer-media-api:${{ github.sha }}
          args: --severity-threshold=high --file=Dockerfile

  scan-dockerfile:
    name: Dockerfile Best Practices
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Run Hadolint
        uses: hadolint/hadolint-action@v3.1.0
        with:
          dockerfile: Dockerfile
          failure-threshold: warning

  scan-dependencies:
    name: Dependency Scanning
    runs-on: ubuntu-latest
    
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: Run govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Run npm audit
        working-directory: ./frontend
        run: |
          npm ci
          npm audit --audit-level=moderate
```

### GitLab CI

Create `.gitlab-ci.yml`:

```yaml
stages:
  - build
  - scan
  - test

variables:
  IMAGE_NAME: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHORT_SHA
  DOCKER_DRIVER: overlay2

build:
  stage: build
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker build -t $IMAGE_NAME .
    - docker push $IMAGE_NAME

trivy-scan:
  stage: scan
  image: aquasec/trivy:latest
  dependencies:
    - build
  script:
    - trivy image --exit-code 0 --format json --output trivy-report.json $IMAGE_NAME
    - trivy image --exit-code 1 --severity CRITICAL,HIGH $IMAGE_NAME
  artifacts:
    reports:
      container_scanning: trivy-report.json
    when: always
  allow_failure: false

snyk-scan:
  stage: scan
  image: snyk/snyk:docker
  dependencies:
    - build
  script:
    - snyk auth $SNYK_TOKEN
    - snyk container test $IMAGE_NAME --severity-threshold=high
  only:
    - main
    - develop

govulncheck:
  stage: scan
  image: golang:1.24
  script:
    - go install golang.org/x/vuln/cmd/govulncheck@latest
    - govulncheck ./...
```

### Jenkins Pipeline

Create `Jenkinsfile`:

```groovy
pipeline {
    agent any
    
    environment {
        IMAGE_NAME = "volunteer-media-api"
        IMAGE_TAG = "${BUILD_NUMBER}"
    }
    
    stages {
        stage('Build') {
            steps {
                script {
                    docker.build("${IMAGE_NAME}:${IMAGE_TAG}")
                }
            }
        }
        
        stage('Security Scan - Trivy') {
            steps {
                script {
                    sh """
                        docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
                        aquasec/trivy:latest image \
                        --exit-code 1 \
                        --severity CRITICAL,HIGH \
                        --format json \
                        --output trivy-report.json \
                        ${IMAGE_NAME}:${IMAGE_TAG}
                    """
                }
            }
            post {
                always {
                    archiveArtifacts artifacts: 'trivy-report.json', allowEmptyArchive: true
                }
            }
        }
        
        stage('Security Scan - Snyk') {
            environment {
                SNYK_TOKEN = credentials('snyk-api-token')
            }
            steps {
                script {
                    sh """
                        docker run --rm -e SNYK_TOKEN=${SNYK_TOKEN} \
                        -v /var/run/docker.sock:/var/run/docker.sock \
                        snyk/snyk:docker test --docker ${IMAGE_NAME}:${IMAGE_TAG} \
                        --severity-threshold=high
                    """
                }
            }
        }
        
        stage('Vulnerability Check - Go') {
            steps {
                script {
                    sh """
                        go install golang.org/x/vuln/cmd/govulncheck@latest
                        govulncheck ./...
                    """
                }
            }
        }
    }
    
    post {
        failure {
            emailext(
                subject: "Security Scan Failed: ${env.JOB_NAME} - ${env.BUILD_NUMBER}",
                body: "Build ${env.BUILD_URL} failed security scan. Please review.",
                to: "security-team@yourdomain.com"
            )
        }
    }
}
```

### CircleCI

Create `.circleci/config.yml`:

```yaml
version: 2.1

orbs:
  snyk: snyk/snyk@1.5.0

jobs:
  build-and-scan:
    docker:
      - image: cimg/go:1.24
    steps:
      - checkout
      
      - setup_remote_docker:
          docker_layer_caching: true
      
      - run:
          name: Build Docker image
          command: |
            docker build -t volunteer-media-api:${CIRCLE_SHA1} .
      
      - run:
          name: Install Trivy
          command: |
            wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | sudo apt-key add -
            echo "deb https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee -a /etc/apt/sources.list.d/trivy.list
            sudo apt-get update
            sudo apt-get install trivy
      
      - run:
          name: Scan image with Trivy
          command: |
            trivy image --exit-code 1 --severity HIGH,CRITICAL volunteer-media-api:${CIRCLE_SHA1}
      
      - snyk/scan:
          docker-image-name: volunteer-media-api:${CIRCLE_SHA1}
          severity-threshold: high

workflows:
  version: 2
  build-scan-deploy:
    jobs:
      - build-and-scan
```

## Best Practices

### 1. Multi-Stage Scanning

Scan at multiple stages of the pipeline:
- **Pre-build**: Scan base images
- **Post-build**: Scan final image
- **Pre-deployment**: Final verification
- **Runtime**: Continuous monitoring

### 2. Fail Fast

Configure scans to fail the build on critical/high vulnerabilities:
```bash
# Trivy
trivy image --exit-code 1 --severity CRITICAL,HIGH image:tag

# Snyk
snyk container test --severity-threshold=high image:tag

# Grype
grype image:tag --fail-on high
```

### 3. Ignore Known False Positives

Create `.trivyignore` file:
```
# Format: CVE-ID
CVE-2023-12345
CVE-2023-67890
```

For Snyk, create `.snyk` file:
```yaml
# Snyk (https://snyk.io) policy file
version: v1.0.0
ignore:
  CVE-2023-12345:
    - '*':
        reason: False positive - not applicable to our use case
        expires: 2025-12-31T00:00:00.000Z
```

### 4. Regular Base Image Updates

Keep base images up to date in Dockerfile:
```dockerfile
# Pin specific version but update regularly
FROM golang:1.24-alpine AS builder
FROM node:20-alpine AS frontend-builder
FROM scratch AS final
```

Check for updates:
```bash
# Check Docker Hub for latest version
docker pull golang:1.24-alpine
docker pull node:20-alpine

# Rebuild with --no-cache to get latest patches
docker build --no-cache -t volunteer-media-api:latest .
```

### 5. Automated Dependency Updates

Use tools like Dependabot or Renovate:

**GitHub Dependabot** (`.github/dependabot.yml`):
```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10

  - package-ecosystem: "npm"
    directory: "/frontend"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10

  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"
```

### 6. Runtime Scanning

Implement runtime scanning in production:
```bash
# Schedule daily scans of running containers
0 2 * * * trivy image $(docker ps --format "{{.Image}}" | head -1) --severity HIGH,CRITICAL
```

### 7. Security Policy Enforcement

Create `security-policy.yml`:
```yaml
# Maximum allowed CVSS score
max_cvss_score: 7.0

# Allowed severities in production
allowed_severities:
  - LOW
  - MEDIUM

# Auto-fix if available
auto_fix: true

# Notification channels
notifications:
  slack: "#security-alerts"
  email: "security-team@yourdomain.com"

# Exemptions (with expiry)
exemptions:
  - cve: CVE-2023-12345
    reason: "False positive - not exploitable in our context"
    expires: 2025-12-31
```

## Reporting and Metrics

### Generate Reports

```bash
# Trivy HTML report
trivy image --format template --template "@contrib/html.tpl" -o report.html image:tag

# JSON for processing
trivy image --format json -o report.json image:tag

# SARIF for GitHub Security
trivy image --format sarif -o report.sarif image:tag
```

### Track Metrics

Key metrics to monitor:
- Number of critical/high vulnerabilities per build
- Time to remediate vulnerabilities
- Percentage of builds passing security scan
- Number of vulnerabilities in production images
- Mean time to patch (MTTP)

### Dashboard Example

Use tools like Grafana to visualize:
```json
{
  "dashboard": {
    "title": "Container Security Dashboard",
    "panels": [
      {
        "title": "Vulnerabilities by Severity",
        "type": "graph"
      },
      {
        "title": "Scan Pass Rate",
        "type": "stat"
      },
      {
        "title": "Time to Remediate",
        "type": "gauge"
      }
    ]
  }
}
```

## Compliance and Auditing

### Compliance Requirements

Document scan results for compliance:
```bash
# Generate compliance report
trivy image --format json volunteer-media-api:latest | \
  jq '{
    scan_date: now|strftime("%Y-%m-%d"),
    image: .ArtifactName,
    vulnerabilities: .Results[].Vulnerabilities | length,
    critical: [.Results[].Vulnerabilities[] | select(.Severity=="CRITICAL")] | length,
    high: [.Results[].Vulnerabilities[] | select(.Severity=="HIGH")] | length
  }' > compliance-report.json
```

### Audit Trail

Maintain scan history:
```bash
# Archive scan results
mkdir -p /audit/scans/$(date +%Y-%m)
trivy image --format json volunteer-media-api:latest > \
  /audit/scans/$(date +%Y-%m)/scan-$(date +%Y%m%d-%H%M%S).json
```

## Troubleshooting

### Common Issues

**Issue**: Scan takes too long
```bash
# Use cached database
trivy image --cache-dir /tmp/trivy-cache image:tag

# Skip database update
trivy image --skip-db-update image:tag
```

**Issue**: False positives
```bash
# Ignore specific CVEs
trivy image --ignorefile .trivyignore image:tag

# Filter by package
trivy image --ignore-unfixed image:tag
```

**Issue**: Network timeouts
```bash
# Increase timeout
trivy image --timeout 10m image:tag

# Use offline database
trivy image --offline-scan image:tag
```

## Resources

- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [Snyk Container Documentation](https://docs.snyk.io/products/snyk-container)
- [Docker Security Best Practices](https://docs.docker.com/develop/security-best-practices/)
- [NIST Container Security Guide](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-190.pdf)
- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)

---

**Document Version**: 1.0  
**Last Updated**: 2025-10-31  
**Owner**: Security Team

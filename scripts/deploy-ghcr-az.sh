#!/usr/bin/env bash
set -euo pipefail

# Usage: 
#   ./scripts/deploy-ghcr-az.sh [options]
#   ./scripts/deploy-ghcr-az.sh --rollback 20260117-abc1234
#   ./scripts/deploy-ghcr-az.sh --list-revisions
#   ./scripts/deploy-ghcr-az.sh --image-tag develop
#
# Options:
#   --image-tag <tag>       Base image tag (default: develop)
#   --rollback <revision>   Deploy a specific revision (format: YYYYMMDD-gitsha)
#   --list-revisions        List available revisions
#   --skip-build            Skip building, deploy existing revision
#
# Environment Variables:
#   REPO: ghcr.io/networkengineer-cloud/go-volunteer-media
#   APP_NAME: ca-volunteer-media-dev
#   RG: rg-volunteer-media-dev

# Defaults
IMAGE_TAG="develop"
REPO="${REPO:-ghcr.io/networkengineer-cloud/go-volunteer-media}"
APP_NAME="${APP_NAME:-ca-volunteer-media-dev}"
RG="${RG:-rg-volunteer-media-dev}"
ROLLBACK_REVISION=""
LIST_REVISIONS=false
SKIP_BUILD=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --image-tag)
      IMAGE_TAG="$2"
      shift 2
      ;;
    --rollback)
      ROLLBACK_REVISION="$2"
      shift 2
      ;;
    --list-revisions)
      LIST_REVISIONS=true
      shift
      ;;
    --skip-build)
      SKIP_BUILD=true
      shift
      ;;
    -h|--help)
      echo "Usage: ./scripts/deploy-ghcr-az.sh [options]"
      echo ""
      echo "Options:"
      echo "  --image-tag <tag>       Base image tag (default: develop)"
      echo "  --rollback <revision>   Deploy a specific revision (format: YYYYMMDD-gitsha)"
      echo "  --list-revisions        List available revisions"
      echo "  --skip-build            Skip building, deploy existing revision"
      echo "  -h, --help              Show this help message"
      echo ""
      echo "Examples:"
      echo "  ./scripts/deploy-ghcr-az.sh --image-tag main"
      echo "  ./scripts/deploy-ghcr-az.sh --rollback 20260117-abc1234"
      echo "  ./scripts/deploy-ghcr-az.sh --list-revisions"
      exit 0
      ;;
    *)
      # Legacy support: first positional arg is image tag
      if [[ -z "${IMAGE_TAG}" ]] || [[ "${IMAGE_TAG}" == "develop" ]]; then
        IMAGE_TAG="$1"
      fi
      shift
      ;;
  esac
done

# Function to list available revisions
list_revisions() {
  echo "Available revisions in ${REPO}:"
  echo "========================================"
  
  # List tags from Docker registry (if gh CLI is available)
  if command -v gh &> /dev/null; then
    echo "Fetching from GitHub Container Registry..."
    gh api -H "Accept: application/vnd.github+json" \
      "/orgs/networkengineer-cloud/packages/container/go-volunteer-media/versions" \
      --jq '.[] | select(.metadata.container.tags[] | test("^[0-9]{8}-")) | .metadata.container.tags[] | select(test("^[0-9]{8}-"))' 2>/dev/null | sort -r | head -20 || echo "Could not fetch from registry"
  fi
  
  echo ""
  echo "Active Azure Container App revisions:"
  echo "========================================"
  az containerapp revision list \
    --name "${APP_NAME}" \
    --resource-group "${RG}" \
    --query "[].{Name:name,Created:properties.createdTime,Active:properties.active,Traffic:properties.trafficWeight,Image:properties.template.containers[0].image}" \
    -o table
}

# Handle --list-revisions
if [[ "${LIST_REVISIONS}" == true ]]; then
  list_revisions
  exit 0
fi

# Generate revision tag (date + git SHA)
if [[ -z "${ROLLBACK_REVISION}" ]]; then
  DATE=$(date +%Y%m%d)
  GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  REVISION_TAG="${DATE}-${GIT_SHA}"
  
  echo "Creating new revision: ${REVISION_TAG}"
else
  REVISION_TAG="${ROLLBACK_REVISION}"
  echo "Rolling back to revision: ${REVISION_TAG}"
  SKIP_BUILD=true
fi

FULL_IMAGE_TAG="${REPO}:${REVISION_TAG}"

# Build and push new image (unless rolling back)
if [[ "${SKIP_BUILD}" == false ]]; then
  echo "Building image ${FULL_IMAGE_TAG} for linux/amd64..."
  docker build --platform linux/amd64 \
    -t "${FULL_IMAGE_TAG}" \
    -t "${REPO}:${IMAGE_TAG}" \
    --label "revision=${REVISION_TAG}" \
    --label "git-sha=${GIT_SHA}" \
    --label "build-date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    .
  
  echo "Pushing revision image to registry..."
  docker push "${FULL_IMAGE_TAG}"
  
  echo "Pushing base tag to registry..."
  docker push "${REPO}:${IMAGE_TAG}"
else
  echo "Skipping build (--skip-build or --rollback specified)"
  
  # Verify the image exists
  if ! docker manifest inspect "${FULL_IMAGE_TAG}" &> /dev/null; then
    echo "Error: Image ${FULL_IMAGE_TAG} not found in registry"
    echo "Available revisions:"
    list_revisions
    exit 1
  fi
fi

echo ""
echo "Updating Azure Container App '${APP_NAME}' to use image ${FULL_IMAGE_TAG}..."

# Generate Azure revision suffix (max 10 chars, alphanumeric and hyphens only)
# Format: MMDD-gitsha (e.g., 0117-abc1234 = 12 chars, or 0117abc123 = 10 chars)
MONTH_DAY=$(echo ${REVISION_TAG} | cut -d'-' -f1 | cut -c5-8)  # MMDD from YYYYMMDD
GIT_SHORT=$(echo ${REVISION_TAG} | cut -d'-' -f2 | cut -c1-6)  # First 6 chars of git SHA
REVISION_SUFFIX="${MONTH_DAY}${GIT_SHORT}"

echo "Using Azure revision suffix: ${REVISION_SUFFIX}"

az containerapp update \
  --name "${APP_NAME}" \
  --resource-group "${RG}" \
  --image "${FULL_IMAGE_TAG}" \
  --revision-suffix "${REVISION_SUFFIX}"

# Give Azure a moment to create the new revision
sleep 5

# Get the most recent revision name
AZURE_REVISION=$(az containerapp revision list \
  --name "${APP_NAME}" \
  --resource-group "${RG}" \
  --query "[0].name" \
  -o tsv || true)

if [[ -n "${AZURE_REVISION}" ]]; then
  echo "Active revision: ${AZURE_REVISION}"
  echo ""
  echo "Verifying revision health..."
  az containerapp revision show \
    --name "${APP_NAME}" \
    --resource-group "${RG}" \
    --revision "${AZURE_REVISION}" \
    --query "{Name:name,Active:properties.active,Replicas:properties.replicas,Health:properties.healthState,Image:properties.template.containers[0].image}" \
    -o table
else
  echo "Warning: Could not retrieve revision information"
fi

echo ""
echo "=========================================="
echo "Deployment finished successfully!"
echo "=========================================="
echo "Revision: ${REVISION_TAG}"
echo "Image: ${FULL_IMAGE_TAG}"
echo ""
echo "To rollback to this revision later, run:"
echo "  ./scripts/deploy-ghcr-az.sh --rollback ${REVISION_TAG}"
echo ""
echo "To list all revisions:"
echo "  ./scripts/deploy-ghcr-az.sh --list-revisions"

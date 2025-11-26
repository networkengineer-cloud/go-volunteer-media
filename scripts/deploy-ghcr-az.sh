#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/deploy-ghcr-az.sh [image-tag]
# Defaults:
#  image-tag: develop
#  REPO: ghcr.io/networkengineer-cloud/go-volunteer-media
#  APP_NAME: ca-volunteer-media-dev
#  RG: rg-volunteer-media-dev

IMAGE_TAG="${1:-develop}"
REPO="${REPO:-ghcr.io/networkengineer-cloud/go-volunteer-media}"
APP_NAME="${APP_NAME:-ca-volunteer-media-dev}"
RG="${RG:-rg-volunteer-media-dev}"

echo "Building image ${REPO}:${IMAGE_TAG}..."
docker build -t "${REPO}:${IMAGE_TAG}" .

echo "Pushing image to registry..."
docker push "${REPO}:${IMAGE_TAG}"

echo "Updating Azure Container App '${APP_NAME}' to use image ${REPO}:${IMAGE_TAG}..."
az containerapp update --name "${APP_NAME}" --resource-group "${RG}" --image "${REPO}:${IMAGE_TAG}"

# Give Azure a moment to create a new revision
sleep 3

# Try to get the most recent revision name (Azure typically returns newest first)
REVISION=$(az containerapp revision list --name "${APP_NAME}" --resource-group "${RG}" --query "[0].name" -o tsv || true)

if [[ -n "${REVISION}" ]]; then
  echo "Restarting revision ${REVISION} to ensure it's active..."
  az containerapp revision restart --name "${APP_NAME}" --resource-group "${RG}" --revision "${REVISION}"
else
  echo "Warning: no revision found to restart. The update command may have failed or there are no revisions yet."
fi

echo "Deployment finished."

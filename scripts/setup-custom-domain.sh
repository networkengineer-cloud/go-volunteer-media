#!/bin/bash
set -e

# Setup Custom Domain for Azure Container App
# Run this after `terraform apply` to configure managed certificates
# Usage: bash scripts/setup-custom-domain.sh [environment]

ENVIRONMENT="${1:-prod}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TERRAFORM_DIR="$SCRIPT_DIR/../terraform/environments/$ENVIRONMENT"

echo "=== Setting up custom domain for $ENVIRONMENT environment ==="

# Check if terraform directory exists
if [ ! -d "$TERRAFORM_DIR" ]; then
  echo "Error: Terraform directory not found: $TERRAFORM_DIR"
  exit 1
fi

# Get values from Terraform outputs
cd "$TERRAFORM_DIR"
RESOURCE_GROUP=$(terraform output -raw resource_group_name)
CONTAINER_APP=$(terraform output -raw container_app_name)
CONTAINER_APP_ENVIRONMENT=$(terraform output -raw container_app_environment_name)
INFRASTRUCTURE_DOMAIN=$(terraform output -raw custom_domain_url | sed 's|https://||')
PUBLIC_DOMAIN=$(terraform output -raw frontend_url | sed 's|https://||')

if [ -z "$INFRASTRUCTURE_DOMAIN" ] || [ "$INFRASTRUCTURE_DOMAIN" == "null" ]; then
  echo "No custom domain configured in Terraform. Exiting."
  exit 0
fi

echo "Resource Group: $RESOURCE_GROUP"
echo "Container App: $CONTAINER_APP"
echo "Container App Environment: $CONTAINER_APP_ENVIRONMENT"
echo "Infrastructure Domain: $INFRASTRUCTURE_DOMAIN"
echo "Public Domain: $PUBLIC_DOMAIN"
echo ""

# Function to check and add hostname with Azure managed certificate
add_hostname_with_cert() {
  local DOMAIN=$1

  echo "=== Processing: $DOMAIN (Azure managed certificate) ==="

  # Check if custom domain already exists
  echo "Checking if custom domain is already configured..."
  EXISTING=$(az containerapp hostname list \
    --name "$CONTAINER_APP" \
    --resource-group "$RESOURCE_GROUP" \
    --query "[?name=='$DOMAIN'].name" -o tsv || true)

  if [ ! -z "$EXISTING" ]; then
    echo "✓ Custom domain already configured: $DOMAIN"

    # Check binding status
    BINDING=$(az containerapp hostname list \
      --name "$CONTAINER_APP" \
      --resource-group "$RESOURCE_GROUP" \
      --query "[?name=='$DOMAIN'].bindingType" -o tsv)

    echo "  Binding Type: $BINDING"

    if [ "$BINDING" == "SniEnabled" ]; then
      echo "✓ Certificate is properly bound"
      return 0
    else
      echo "⚠ Certificate binding is disabled, will re-add..."
      az containerapp hostname delete \
        --hostname "$DOMAIN" \
        --resource-group "$RESOURCE_GROUP" \
        --name "$CONTAINER_APP" \
        --yes
    fi
  fi

  # Step 1: Add hostname to the container app (without certificate)
  echo ""
  echo "Step 1: Adding hostname to container app..."
  az containerapp hostname add \
    --hostname "$DOMAIN" \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CONTAINER_APP"

  # Step 2: Bind managed certificate to the hostname
  echo ""
  echo "Step 2: Binding managed certificate..."
  echo "This may take 5-15 minutes for Azure to provision the certificate..."
  az containerapp hostname bind \
    --hostname "$DOMAIN" \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CONTAINER_APP" \
    --environment "$CONTAINER_APP_ENVIRONMENT" \
    --validation-method CNAME

  echo ""
  echo "✓ Custom domain setup initiated for $DOMAIN!"
  echo ""
}

# Function to add hostname only (no Azure cert — TLS handled by Cloudflare proxy)
add_hostname_cloudflare() {
  local DOMAIN=$1

  echo "=== Processing: $DOMAIN (Cloudflare proxied — no Azure cert needed) ==="

  # Check if custom domain already exists
  echo "Checking if custom domain is already configured..."
  EXISTING=$(az containerapp hostname list \
    --name "$CONTAINER_APP" \
    --resource-group "$RESOURCE_GROUP" \
    --query "[?name=='$DOMAIN'].name" -o tsv || true)

  if [ ! -z "$EXISTING" ]; then
    echo "✓ Custom domain already configured: $DOMAIN"

    BINDING=$(az containerapp hostname list \
      --name "$CONTAINER_APP" \
      --resource-group "$RESOURCE_GROUP" \
      --query "[?name=='$DOMAIN'].bindingType" -o tsv)

    echo "  Binding Type: $BINDING"
    echo "  (Disabled is expected — Cloudflare handles TLS)"
    return 0
  fi

  echo ""
  echo "Adding hostname to container app (Cloudflare handles TLS)..."
  az containerapp hostname add \
    --hostname "$DOMAIN" \
    --resource-group "$RESOURCE_GROUP" \
    --name "$CONTAINER_APP"

  echo ""
  echo "✓ Hostname added for $DOMAIN (TLS via Cloudflare proxy)"
  echo ""
}

# Add infrastructure domain (prd.myhaws.org) — Azure managed certificate
add_hostname_with_cert "$INFRASTRUCTURE_DOMAIN"

# Add public domain (www.myhaws.org) — Cloudflare proxied, no Azure cert
if [ "$PUBLIC_DOMAIN" != "$INFRASTRUCTURE_DOMAIN" ]; then
  add_hostname_cloudflare "$PUBLIC_DOMAIN"
fi

echo ""
echo "✓ All custom domains configured!"
echo ""
echo "Next steps:"
echo "1. Wait 5-15 minutes for Azure to provision managed certificates"
echo "2. Verify status: az containerapp hostname list --name $CONTAINER_APP --resource-group $RESOURCE_GROUP -o table"
echo "3. Ensure Cloudflare proxy (orange cloud) is enabled for $PUBLIC_DOMAIN"
echo "4. Set Cloudflare SSL/TLS mode to 'Full' for origin connections"
echo "5. Access your app at:"
echo "   - https://$INFRASTRUCTURE_DOMAIN (direct)"
echo "   - https://$PUBLIC_DOMAIN (via Cloudflare)"

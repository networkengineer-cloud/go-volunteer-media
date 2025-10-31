#!/bin/bash
# Initial setup script for Azure infrastructure
# Run this script once to create the Terraform state storage

set -e

echo "🚀 Setting up Azure Infrastructure for Volunteer Media Platform"
echo "==============================================================="
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo "❌ Azure CLI is not installed. Please install it first:"
    echo "   https://learn.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check if Terraform is installed
if ! command -v terraform &> /dev/null; then
    echo "❌ Terraform is not installed. Please install it first:"
    echo "   https://www.terraform.io/downloads"
    exit 1
fi

echo "✅ Azure CLI and Terraform are installed"
echo ""

# Check if logged in to Azure
echo "🔐 Checking Azure login status..."
if ! az account show &> /dev/null; then
    echo "❌ Not logged in to Azure. Logging in now..."
    az login
else
    echo "✅ Already logged in to Azure"
    CURRENT_SUBSCRIPTION=$(az account show --query name -o tsv)
    echo "   Current subscription: $CURRENT_SUBSCRIPTION"
fi

echo ""
echo "📦 Creating Terraform state storage..."
echo ""

# Variables for state storage
STATE_RESOURCE_GROUP="rg-terraform-state"
STATE_STORAGE_ACCOUNT="sttfstatevolunteer"
STATE_CONTAINER="tfstate"
LOCATION="eastus"

# Create resource group for state storage
echo "Creating resource group: $STATE_RESOURCE_GROUP"
az group create \
    --name "$STATE_RESOURCE_GROUP" \
    --location "$LOCATION" \
    --tags "ManagedBy=Manual" "Purpose=TerraformState" \
    --output none

echo "✅ Resource group created"

# Create storage account for state
echo "Creating storage account: $STATE_STORAGE_ACCOUNT"
az storage account create \
    --name "$STATE_STORAGE_ACCOUNT" \
    --resource-group "$STATE_RESOURCE_GROUP" \
    --location "$LOCATION" \
    --sku Standard_LRS \
    --encryption-services blob \
    --https-only true \
    --min-tls-version TLS1_2 \
    --allow-blob-public-access false \
    --output none

echo "✅ Storage account created"

# Create blob container for state files
echo "Creating blob container: $STATE_CONTAINER"
az storage container create \
    --name "$STATE_CONTAINER" \
    --account-name "$STATE_STORAGE_ACCOUNT" \
    --auth-mode login \
    --output none

echo "✅ Blob container created"

echo ""
echo "✅ Terraform state storage setup complete!"
echo ""
echo "📝 Next steps:"
echo "   1. Copy terraform/environments/prod/terraform.tfvars.example to terraform.tfvars"
echo "   2. Edit terraform.tfvars with your configuration"
echo "   3. Create SendGrid account and get API key:"
echo "      https://signup.sendgrid.com/"
echo "   4. Generate JWT secret:"
echo "      openssl rand -base64 32"
echo "   5. Build and push your container image to GHCR"
echo "   6. Run terraform init in terraform/environments/prod/"
echo "   7. Run terraform plan to preview changes"
echo "   8. Run terraform apply to deploy infrastructure"
echo ""
echo "📚 Documentation: See terraform/README.md for detailed instructions"
echo ""

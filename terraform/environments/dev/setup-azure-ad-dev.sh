#!/bin/bash
# Setup Azure AD Application for Dev Environment
# This script creates a separate Azure AD app for dev with OIDC federated credentials
# Run this script with: bash terraform/environments/dev/setup-azure-ad-dev.sh

set -e

echo "🚀 Setting up Azure AD Application for Dev Environment..."
echo ""

# Configuration
APP_NAME="HCP-Terraform-OIDC-Dev"
HCP_ORG="Networkengineer"
HCP_WORKSPACE="volunteer-app-dev"
GITHUB_ORG="networkengineer-cloud"
GITHUB_REPO="go-volunteer-media"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}Step 1: Checking Azure CLI authentication...${NC}"
if ! az account show &> /dev/null; then
    echo "❌ Not logged in to Azure CLI. Please run: az login"
    exit 1
fi
echo "✅ Azure CLI authenticated"
echo ""

# Get Azure subscription info
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
TENANT_ID=$(az account show --query tenantId -o tsv)
SUBSCRIPTION_NAME=$(az account show --query name -o tsv)

echo -e "${BLUE}Azure Subscription Information:${NC}"
echo "  Subscription: $SUBSCRIPTION_NAME"
echo "  Subscription ID: $SUBSCRIPTION_ID"
echo "  Tenant ID: $TENANT_ID"
echo ""

# Check if app already exists
echo -e "${BLUE}Step 2: Checking if Azure AD app already exists...${NC}"
EXISTING_APP_ID=$(az ad app list --display-name "$APP_NAME" --query "[0].appId" -o tsv)

if [ -n "$EXISTING_APP_ID" ]; then
    echo -e "${YELLOW}⚠️  App '$APP_NAME' already exists with ID: $EXISTING_APP_ID${NC}"
    read -p "Do you want to use the existing app? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        APP_ID=$EXISTING_APP_ID
        echo "✅ Using existing app"
    else
        echo "❌ Aborting. Please delete the existing app or use a different name."
        exit 1
    fi
else
    echo -e "${BLUE}Creating new Azure AD app: $APP_NAME${NC}"
    APP_ID=$(az ad app create \
        --display-name "$APP_NAME" \
        --sign-in-audience AzureADMyOrg \
        --query appId -o tsv)
    echo "✅ Created app with ID: $APP_ID"
fi
echo ""

# Create service principal if it doesn't exist
echo -e "${BLUE}Step 3: Creating service principal...${NC}"
SP_EXISTS=$(az ad sp list --filter "appId eq '$APP_ID'" --query "[0].id" -o tsv)

if [ -n "$SP_EXISTS" ]; then
    echo "✅ Service principal already exists"
    SP_OBJECT_ID=$SP_EXISTS
else
    az ad sp create --id "$APP_ID" > /dev/null
    SP_OBJECT_ID=$(az ad sp list --filter "appId eq '$APP_ID'" --query "[0].id" -o tsv)
    echo "✅ Created service principal"
    sleep 5  # Wait for propagation
fi
echo ""

# Assign Contributor role to resource group (will be created by Terraform)
echo -e "${BLUE}Step 4: Assigning Azure RBAC permissions...${NC}"
RESOURCE_GROUP="rg-volunteer-media-dev"

# Check if resource group exists
if az group show --name "$RESOURCE_GROUP" &> /dev/null; then
    echo "  Resource group '$RESOURCE_GROUP' exists"
    
    # Assign Contributor role
    az role assignment create \
        --role "Contributor" \
        --assignee-object-id "$SP_OBJECT_ID" \
        --scope "/subscriptions/$SUBSCRIPTION_ID/resourceGroups/$RESOURCE_GROUP" \
        --only-show-errors > /dev/null 2>&1 || true
    echo "✅ Assigned Contributor role to resource group"
else
    echo -e "${YELLOW}  Resource group doesn't exist yet (will be created by Terraform)${NC}"
    echo "  Assigning Contributor role at subscription level..."
    
    az role assignment create \
        --role "Contributor" \
        --assignee-object-id "$SP_OBJECT_ID" \
        --scope "/subscriptions/$SUBSCRIPTION_ID" \
        --only-show-errors > /dev/null 2>&1 || true
    echo "✅ Assigned Contributor role at subscription level"
    echo -e "${YELLOW}  ⚠️  Note: This is broader than needed. After first deployment, reassign to resource group only.${NC}"
fi
echo ""

# Configure federated credential for HCP Terraform
echo -e "${BLUE}Step 5: Configuring federated credential for HCP Terraform...${NC}"

FEDERATED_CRED_NAME="hcp-terraform-oidc"
SUBJECT="organization:$HCP_ORG:project:*:workspace:$HCP_WORKSPACE:run_phase:*"

# Check if federated credential already exists
EXISTING_CRED=$(az ad app federated-credential list --id "$APP_ID" \
    --query "[?name=='$FEDERATED_CRED_NAME'].name" -o tsv)

if [ -n "$EXISTING_CRED" ]; then
    echo -e "${YELLOW}  Federated credential already exists. Deleting and recreating...${NC}"
    az ad app federated-credential delete \
        --id "$APP_ID" \
        --federated-credential-id "$FEDERATED_CRED_NAME" > /dev/null 2>&1 || true
    sleep 2
fi

az ad app federated-credential create \
    --id "$APP_ID" \
    --parameters "{
        \"name\": \"$FEDERATED_CRED_NAME\",
        \"issuer\": \"https://app.terraform.io\",
        \"subject\": \"$SUBJECT\",
        \"description\": \"Federated credential for HCP Terraform dev workspace\",
        \"audiences\": [\"api://AzureADTokenExchange\"]
    }" > /dev/null

echo "✅ Configured federated credential for HCP Terraform"
echo "   Subject: $SUBJECT"
echo ""

# Configure federated credential for GitHub Actions (develop branch)
echo -e "${BLUE}Step 6: Configuring federated credential for GitHub Actions...${NC}"

GITHUB_CRED_NAME="github-actions-develop"
GITHUB_SUBJECT="repo:$GITHUB_ORG/$GITHUB_REPO:ref:refs/heads/develop"

# Check if GitHub credential already exists
EXISTING_GITHUB_CRED=$(az ad app federated-credential list --id "$APP_ID" \
    --query "[?name=='$GITHUB_CRED_NAME'].name" -o tsv)

if [ -n "$EXISTING_GITHUB_CRED" ]; then
    echo -e "${YELLOW}  GitHub federated credential already exists. Deleting and recreating...${NC}"
    az ad app federated-credential delete \
        --id "$APP_ID" \
        --federated-credential-id "$GITHUB_CRED_NAME" > /dev/null 2>&1 || true
    sleep 2
fi

az ad app federated-credential create \
    --id "$APP_ID" \
    --parameters "{
        \"name\": \"$GITHUB_CRED_NAME\",
        \"issuer\": \"https://token.actions.githubusercontent.com\",
        \"subject\": \"$GITHUB_SUBJECT\",
        \"description\": \"Federated credential for GitHub Actions develop branch\",
        \"audiences\": [\"api://AzureADTokenExchange\"]
    }" > /dev/null

echo "✅ Configured federated credential for GitHub Actions"
echo "   Subject: $GITHUB_SUBJECT"
echo ""

# Summary
echo -e "${GREEN}✅ Azure AD Application Setup Complete!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BLUE}Next Steps:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "1️⃣  Add GitHub Secret: AZURE_CLIENT_ID_DEV"
echo "   Value: $APP_ID"
echo "   Command:"
echo "   gh secret set AZURE_CLIENT_ID_DEV --body \"$APP_ID\""
echo ""
echo "2️⃣  Configure HCP Terraform Workspace: $HCP_WORKSPACE"
echo "   Go to: https://app.terraform.io/app/$HCP_ORG/workspaces/$HCP_WORKSPACE/variables"
echo ""
echo "   Add these Environment Variables:"
echo "   • ARM_CLIENT_ID = $APP_ID"
echo "   • ARM_TENANT_ID = $TENANT_ID"
echo "   • ARM_SUBSCRIPTION_ID = $SUBSCRIPTION_ID"
echo "   • ARM_USE_OIDC = true"
echo ""
echo "3️⃣  Add GitHub Secrets for the dev environment:"
echo "   • DEV_RESEND_API_KEY - Get from https://resend.com/api-keys"
echo "   • DEV_JWT_SECRET - Generate with: openssl rand -base64 32"
echo "   • DEV_CONTAINER_IMAGE - e.g., ghcr.io/$GITHUB_ORG/volunteer-media:develop"
echo ""
echo "   Quick add with GitHub CLI:"
echo "   gh secret set AZURE_CLIENT_ID_DEV --body \"$APP_ID\""
echo "   gh secret set DEV_RESEND_API_KEY --body \"<your-resend-key>\""
echo "   gh secret set DEV_JWT_SECRET --body \"\$(openssl rand -base64 32)\""
echo "   gh secret set DEV_CONTAINER_IMAGE --body \"ghcr.io/$GITHUB_ORG/volunteer-media:develop\""
echo ""
echo "4️⃣  Create GitHub Environment:"
echo "   Go to: https://github.com/$GITHUB_ORG/$GITHUB_REPO/settings/environments"
echo "   Create environment named: development"
echo ""
echo "5️⃣  Deploy via GitHub Actions:"
echo "   Go to: https://github.com/$GITHUB_ORG/$GITHUB_REPO/actions"
echo "   Select: Terraform Deploy - Development"
echo "   Run workflow → Action: plan (then apply)"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}Setup Information Summary:${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Azure AD App Name: $APP_NAME"
echo "Application ID: $APP_ID"
echo "Tenant ID: $TENANT_ID"
echo "Subscription ID: $SUBSCRIPTION_ID"
echo "HCP Org: $HCP_ORG"
echo "HCP Workspace: $HCP_WORKSPACE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Save to file for reference
OUTPUT_FILE="azure-ad-dev-setup.txt"
cat > "$OUTPUT_FILE" <<EOL
Azure AD Application Setup - Development Environment
Generated: $(date)

Application Name: $APP_NAME
Application (Client) ID: $APP_ID
Tenant ID: $TENANT_ID
Subscription ID: $SUBSCRIPTION_ID
Service Principal Object ID: $SP_OBJECT_ID

HCP Terraform Configuration:
Organization: $HCP_ORG
Workspace: $HCP_WORKSPACE

GitHub Repository: $GITHUB_ORG/$GITHUB_REPO

Federated Credentials:
1. HCP Terraform: $SUBJECT
2. GitHub Actions: $GITHUB_SUBJECT

GitHub Secret to Add:
AZURE_CLIENT_ID_DEV=$APP_ID

HCP Terraform Environment Variables:
ARM_CLIENT_ID=$APP_ID
ARM_TENANT_ID=$TENANT_ID
ARM_SUBSCRIPTION_ID=$SUBSCRIPTION_ID
ARM_USE_OIDC=true
EOL

echo -e "${GREEN}📄 Configuration saved to: $OUTPUT_FILE${NC}"
echo ""

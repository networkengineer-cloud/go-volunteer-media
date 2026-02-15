# Federated Credentials Setup Guide

This guide explains how to set up federated credentials (OIDC) for secure, passwordless authentication between HCP Terraform, GitHub Actions, and Azure.

## What are Federated Credentials?

Federated credentials use OpenID Connect (OIDC) to authenticate services without storing long-lived secrets. Instead of using passwords or access keys, services exchange short-lived tokens.

### Benefits

✅ **No Secrets to Manage**: No passwords, API keys, or access keys to rotate  
✅ **Short-Lived Tokens**: Tokens expire after minutes, not months  
✅ **Audit Trail**: All authentication logged in Azure AD  
✅ **Better Security**: Eliminates secret sprawl and theft risk  
✅ **Zero Trust**: Each request authenticated independently  

## Architecture

```
┌─────────────────┐
│  HCP Terraform  │
│   (Runs plan)   │
└────────┬────────┘
         │ 1. Request token
         │    with workload identity
         ▼
┌─────────────────┐
│   Azure AD      │ 2. Verify identity
│  (Identity      │    against federated
│   Provider)     │    credential policy
└────────┬────────┘
         │ 3. Issue short-lived
         │    access token
         ▼
┌─────────────────┐
│ Azure Resources │ 4. Access granted
│ (Container Apps,│    with RBAC permissions
│  PostgreSQL)    │
└─────────────────┘
```

## Setup Steps

### 1. Register Azure AD Application

This creates the identity that HCP Terraform will use:

```bash
# Create app registration
az ad app create \
  --display-name "HCP-Terraform-OIDC" \
  --sign-in-audience AzureADMyOrg

# Get the Application (client) ID
APP_ID=$(az ad app list --display-name "HCP-Terraform-OIDC" --query "[0].appId" -o tsv)
echo "Application ID: $APP_ID"
```

### 2. Create Service Principal

Service principals are the identity Azure uses for RBAC:

```bash
# Create service principal
az ad sp create --id $APP_ID

# Get Service Principal Object ID
SP_OBJECT_ID=$(az ad sp list --display-name "HCP-Terraform-OIDC" --query "[0].id" -o tsv)
echo "Service Principal ID: $SP_OBJECT_ID"
```

### 3. Assign Azure Permissions

Grant the service principal access to your subscription:

```bash
# Get subscription ID
SUBSCRIPTION_ID=$(az account show --query id -o tsv)

# Assign Contributor role (for creating/managing resources)
az role assignment create \
  --role "Contributor" \
  --assignee-object-id $SP_OBJECT_ID \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --assignee-principal-type "ServicePrincipal"

# Assign User Access Administrator role (for creating role assignments)
# This is required for RBAC-based Key Vault and other role assignments
az role assignment create \
  --role "User Access Administrator" \
  --assignee-object-id $SP_OBJECT_ID \
  --scope "/subscriptions/$SUBSCRIPTION_ID" \
  --assignee-principal-type "ServicePrincipal"

# Verify assignments
az role assignment list \
  --assignee $APP_ID \
  --output table
```

**Why Both Roles?**
- **Contributor**: Creates/manages Azure resources (VMs, databases, etc.)
- **User Access Administrator**: Assigns RBAC roles (Key Vault RBAC, Storage RBAC, etc.)

**Security Note:** User Access Administrator is a privileged role. For production:
- Scope it to specific resource groups instead of the entire subscription
- Or use a custom role with only `Microsoft.Authorization/roleAssignments/write` permission

### 4. Configure Federated Credential for HCP Terraform

This tells Azure AD to trust HCP Terraform's identity tokens:

```bash
# Get your HCP Terraform organization name
HCP_ORG="Networkengineer"  # Replace with your org name

# Create federated credential
az ad app federated-credential create \
  --id $APP_ID \
  --parameters '{
    "name": "hcp-terraform-oidc",
    "issuer": "https://app.terraform.io",
    "subject": "organization:'$HCP_ORG':project:*:workspace:volunteer-app:run_phase:*",
    "description": "Federated credential for HCP Terraform",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

**Subject Pattern Explanation:**
- `organization:volunteer-media`: Your HCP Terraform org
- `project:*`: Any project (or specify project name)
- `workspace:*`: Any workspace (or specify workspace name)
- `run_phase:*`: Any phase (plan, apply, etc.)

### 5. Configure HCP Terraform Workspace

Add these **Environment Variables** in your HCP Terraform workspace:

| Variable | Value | Description |
|----------|-------|-------------|
| `ARM_CLIENT_ID` | `$APP_ID` | Azure Application ID |
| `ARM_TENANT_ID` | Your tenant ID | Get with `az account show --query tenantId -o tsv` |
| `ARM_SUBSCRIPTION_ID` | Your subscription ID | Get with `az account show --query id -o tsv` |
| `ARM_USE_OIDC` | `true` | Enable OIDC authentication |

### 6. Test the Connection

```bash
cd terraform/environments/prod

# Login to HCP Terraform
terraform login

# Initialize
terraform init

# Test with a plan (should authenticate via OIDC)
terraform plan
```

You should see the plan execute without any authentication errors!

## GitHub Actions Setup

For GitHub Actions to use the same federated credential:

### 1. Create Federated Credential for GitHub

```bash
# Get your GitHub org and repo
GITHUB_ORG="networkengineer-cloud"
GITHUB_REPO="go-volunteer-media"

# Create federated credential for GitHub Actions
az ad app federated-credential create \
  --id $APP_ID \
  --parameters '{
    "name": "github-actions-oidc",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'$GITHUB_ORG'/'$GITHUB_REPO':ref:refs/heads/main",
    "description": "Federated credential for GitHub Actions",
    "audiences": ["api://AzureADTokenExchange"]
  }'

# For pull requests, add another credential:
az ad app federated-credential create \
  --id $APP_ID \
  --parameters '{
    "name": "github-actions-oidc-pr",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:'$GITHUB_ORG'/'$GITHUB_REPO':pull_request",
    "description": "Federated credential for GitHub Actions PRs",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

### 2. Add GitHub Secrets

Add these secrets to your GitHub repository:

```bash
# Get the values
echo "AZURE_CLIENT_ID: $APP_ID"
echo "AZURE_TENANT_ID: $(az account show --query tenantId -o tsv)"
echo "AZURE_SUBSCRIPTION_ID: $(az account show --query id -o tsv)"
```

In GitHub repo → Settings → Secrets → Actions:
- `AZURE_CLIENT_ID`: Your app ID
- `AZURE_TENANT_ID`: Your tenant ID
- `AZURE_SUBSCRIPTION_ID`: Your subscription ID
- `TF_API_TOKEN`: Your HCP Terraform token

### 3. Workflow Configuration

The workflow uses these permissions to get OIDC tokens:

```yaml
permissions:
  id-token: write  # Required for OIDC
  contents: read
  pull-requests: write
```

## Troubleshooting

### Error: "AADSTS70021: No matching federated identity record found"

**Cause**: Subject claim doesn't match federated credential configuration

**Fix**: Verify the subject pattern matches your HCP Terraform organization:
```bash
# List federated credentials
az ad app federated-credential list --id $APP_ID

# Check the subject matches your org name
```

### Error: "AuthorizationFailed: The client does not have authorization"

**Cause**: Service principal doesn't have sufficient RBAC permissions

**Fix**: Verify role assignment:
```bash
az role assignment list --assignee $APP_ID --output table
```

### Error: "Could not retrieve token from local cache"

**Cause**: Not logged into HCP Terraform

**Fix**: Login to HCP Terraform:
```bash
terraform login
```

### Testing OIDC Authentication

Test if Azure accepts your OIDC token:

```bash
# In HCP Terraform run, check environment
export ARM_USE_OIDC=true
export ARM_CLIENT_ID="your-app-id"
export ARM_TENANT_ID="your-tenant-id"
export ARM_SUBSCRIPTION_ID="your-subscription-id"

# Try to list resources (should work without password)
terraform plan
```

## Security Considerations

### Token Lifetime

- **OIDC tokens expire**: Default 1 hour
- **Automatic refresh**: Terraform requests new tokens as needed
- **No token storage**: Tokens never written to disk

### Scope Restrictions

For production, restrict the subject claim:

```bash
# Restrict to specific workspace
"subject": "organization:volunteer-media:project:*:workspace:volunteer-media-prod:run_phase:*"

# Restrict to specific GitHub branch
"subject": "repo:org/repo:ref:refs/heads/main"
```

### Least Privilege RBAC

Instead of Contributor, create a custom role:

```bash
az role definition create --role-definition '{
  "Name": "Terraform Deployer",
  "Description": "Can deploy Terraform resources",
  "Actions": [
    "Microsoft.ContainerInstance/*/read",
    "Microsoft.ContainerInstance/*/write",
    "Microsoft.DBforPostgreSQL/*/read",
    "Microsoft.DBforPostgreSQL/*/write",
    "Microsoft.Storage/*/read",
    "Microsoft.Storage/*/write",
    "Microsoft.KeyVault/*/read",
    "Microsoft.KeyVault/*/write"
  ],
  "AssignableScopes": ["/subscriptions/'$SUBSCRIPTION_ID'"]
}'
```

## Additional Resources

- [Azure Workload Identity Federation](https://docs.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation)
- [HCP Terraform Dynamic Credentials](https://developer.hashicorp.com/terraform/cloud-docs/workspaces/dynamic-provider-credentials)
- [GitHub Actions OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- [Azure Terraform Provider OIDC](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_oidc)

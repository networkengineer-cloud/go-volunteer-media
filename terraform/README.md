# Go Volunteer Media - Terraform Infrastructure

This directory contains the Terraform infrastructure as code (IaC) for deploying the Go Volunteer Media application to Azure using **HCP Terraform** (formerly Terraform Cloud) for state management.

## Architecture

- **Azure Container Apps**: Serverless container hosting with auto-scaling
- **PostgreSQL Flexible Server**: Managed database (Burstable B1ms tier)
- **Azure Blob Storage**: Image upload storage
- **SendGrid**: Email service (SMTP - free tier)
- **GitHub Container Registry**: Container image storage (free)
- **Application Insights**: Monitoring and observability
- **Key Vault**: Secrets management
- **HCP Terraform**: Remote state management and collaboration (free tier)

## Prerequisites

1. **Azure CLI**: Install from https://docs.microsoft.com/en-us/cli/azure/install-azure-cli
2. **Terraform**: Version 1.5.0 or later
3. **Azure Subscription**: Active subscription with appropriate permissions
4. **GitHub Account**: For GitHub Container Registry (GHCR)
5. **SendGrid Account**: Free tier (100 emails/day)
6. **HCP Terraform Account**: Sign up at https://app.terraform.io (free tier)

## Project Structure

```
terraform/
├── environments/
│   ├── dev/
│   │   ├── main.tf           # Development environment configuration
│   │   ├── variables.tf      # Environment-specific variables
│   │   ├── terraform.tfvars  # Variable values (gitignored)
│   │   └── backend.tf        # Remote state configuration
│   └── prod/
│       └── ... (same structure)
├── modules/
│   ├── container-app/        # Container Apps module
│   ├── postgresql/           # PostgreSQL module
│   ├── storage/              # Blob Storage module
│   ├── monitoring/           # Application Insights module
│   └── networking/           # VNet and NSG module
└── shared/
    ├── naming.tf             # Naming conventions
    └── tags.tf               # Common tags
```

## Initial Setup

### 1. Create HCP Terraform Account

1. Sign up at https://app.terraform.io
2. Create a new organization: `Networkengineer` (already exists)
3. Create workspaces:
   - Production: `volunteer-app`
   - Development: `volunteer-app-dev`
   - Execution Mode: **Remote** (API-driven workflow)
   - VCS Integration: Not used (GitHub Actions manages deployments)

### 2. Configure Federated Credentials for Azure

**In Azure Portal:**

1. **Register an App in Azure AD (Microsoft Entra ID)**:
   ```bash
   az ad app create --display-name "Terraform-HCP-OIDC"
   ```

2. **Create a Service Principal**:
   ```bash
   APP_ID=$(az ad app list --display-name "Terraform-HCP-OIDC" --query "[0].appId" -o tsv)
   az ad sp create --id $APP_ID
   ```

3. **Assign Contributor Role**:
   ```bash
   SUBSCRIPTION_ID=$(az account show --query id -o tsv)
   SP_ID=$(az ad sp list --display-name "Terraform-HCP-OIDC" --query "[0].id" -o tsv)
   
   az role assignment create \
     --role "Contributor" \
     --assignee-object-id $SP_ID \
     --scope "/subscriptions/$SUBSCRIPTION_ID"
   ```

4. **Configure Federated Credential for HCP Terraform**:
   ```bash
   az ad app federated-credential create \
     --id $APP_ID \
     --parameters '{
       "name": "terraform-cloud-oidc",
       "issuer": "https://app.terraform.io",
       "subject": "organization:YOUR_ORG_NAME:project:*:workspace:*:run_phase:*",
       "description": "HCP Terraform OIDC",
       "audiences": ["api://AzureADTokenExchange"]
     }'
   ```

**In HCP Terraform Workspace:**

1. Go to workspace → Variables
2. Add **Environment Variables**:
   - `ARM_CLIENT_ID`: Your Azure App (client) ID
   - `ARM_SUBSCRIPTION_ID`: Your Azure subscription ID
   - `ARM_TENANT_ID`: Your Azure tenant ID
   - `ARM_USE_OIDC`: `true`

3. Add **Terraform Variables** (mark as sensitive):
   - `sendgrid_api_key`: Your SendGrid API key
   - `jwt_secret`: Generate with `openssl rand -base64 32`
   - `owner_email`: Admin email address
   - `container_image`: Your GHCR image URL

### 3. Configure GitHub Actions Integration

Add these secrets to your GitHub repository:

1. **HCP Terraform Token**:
   - Go to HCP Terraform → User Settings → Tokens
   - Create API token
   - Add to GitHub as `TF_API_TOKEN`

2. **Azure OIDC Credentials** (for GitHub Actions):
   ```bash
   # Get values
   echo "AZURE_CLIENT_ID: $APP_ID"
   echo "AZURE_TENANT_ID: $(az account show --query tenantId -o tsv)"
   echo "AZURE_SUBSCRIPTION_ID: $SUBSCRIPTION_ID"
   ```
   
   Add to GitHub Secrets:
   - `AZURE_CLIENT_ID`
   - `AZURE_TENANT_ID`
   - `AZURE_SUBSCRIPTION_ID`

3. **Application Secrets**:
   - `SENDGRID_API_KEY`: From SendGrid portal
   - `JWT_SECRET`: Generate with `openssl rand -base64 32`
   - `OWNER_EMAIL`: Admin email
   - `CONTAINER_IMAGE`: Your GHCR image URL

### 4. Configure SendGrid (Free Tier)

```bash
# Sign up at https://signup.sendgrid.com
# Free tier: 100 emails/day (forever free)

# After signup:
# 1. Verify sender identity (email or domain)
# 2. Create API key: Settings → API Keys → Create API Key
# 3. Add API key to HCP Terraform workspace variables
```

### 5. Verify Backend Configuration

The backend is already configured for HCP Terraform:

**Production** (`terraform/environments/prod/backend.tf`):
```hcl
cloud {
  organization = "Networkengineer"
  workspaces {
    name = "volunteer-app"
  }
}
```

**Development** (`terraform/environments/dev/backend.tf`):
```hcl
cloud {
  organization = "Networkengineer"
  workspaces {
    name = "volunteer-app-dev"
  }
}
```

### 6. Initialize Terraform

```bash
cd terraform/environments/prod

# Login to HCP Terraform
terraform login

# Initialize
terraform init
```

## Deployment

### Via HCP Terraform (Recommended)

HCP Terraform provides a web UI for managing runs:

1. **Automatic Runs** (with VCS integration):
   - Push to main branch
   - HCP Terraform automatically triggers plan
   - Review plan in UI
   - Approve to apply

2. **Manual Runs**:
   - Go to workspace in HCP Terraform UI
   - Click "Actions" → "Start new run"
   - Review plan
   - Confirm and apply

### Via CLI

```bash
cd terraform/environments/prod

# Plan
terraform plan

# Apply
terraform apply
```

### Via GitHub Actions (CI/CD)

The workflow automatically runs on:
- **Pull Requests**: Runs `terraform plan` and comments on PR
- **Push to main**: Runs `terraform apply` automatically

## Configuration

### Required Variables

Variables are configured in the HCP Terraform workspace UI (no `terraform.tfvars` file needed):

**Terraform Variables** (in HCP Terraform workspace):
```
location = "eastus2"
environment = "prod"
sendgrid_api_key = "SG.xxx"  # Mark as sensitive
jwt_secret = "your-secret"   # Mark as sensitive
owner_email = "admin@example.com"
container_image = "ghcr.io/your-org/app:latest"
```

**Environment Variables** (in HCP Terraform workspace):
```
ARM_CLIENT_ID = "azure-app-id"
ARM_SUBSCRIPTION_ID = "azure-subscription-id"
ARM_TENANT_ID = "azure-tenant-id"
ARM_USE_OIDC = "true"
```

## Outputs

After successful deployment, Terraform will output:

- Container App URL
- Database connection string
- Storage account name
- Application Insights connection string
- Key Vault name

## Cost Estimation

**Monthly Costs (Production):**
- Azure Container Apps: $5-8/month (consumption tier)
- PostgreSQL Flexible Server: $10/month (B1ms)
- Azure Blob Storage: $1-2/month
- Application Insights: Free tier (<5GB)
- SendGrid: **Free tier** (100 emails/day forever)
- HCP Terraform: **Free tier** (up to 500 resources)
- **Total: ~$16-20/month**

## Benefits of HCP Terraform

1. **Remote State Management**: Secure, encrypted state storage
2. **State Locking**: Automatic state locking prevents conflicts
3. **Collaboration**: Team access with RBAC
4. **Run History**: Full audit trail of all changes
5. **Cost Estimation**: Preview costs before applying
6. **VCS Integration**: Automatic runs from GitHub
7. **Web UI**: Visual workspace management
8. **Free Tier**: Up to 500 resources under management
9. **Federated Credentials**: No long-lived secrets needed

## Security Best Practices

1. **Federated Credentials (OIDC)**: 
   - No client secrets stored
   - Short-lived tokens only
   - Azure AD authenticates via OIDC
   
2. **HCP Terraform Security**:
   - Encrypted state storage
   - Sensitive variables marked and encrypted
   - Audit logs for all operations
   - Team-based access control

3. **Azure Security**:
   - Managed identities where possible
   - All data encrypted at rest and in transit
   - Private endpoints for database
   - Network isolation with VNets
   - RBAC with least privilege

4. **GitHub Actions Security**:
   - OIDC for Azure authentication
   - No long-lived credentials in secrets
   - HCP Terraform token rotation supported

## Monitoring

Access monitoring dashboards:

```bash
# Application Insights
az portal show --resource-id $(terraform output -raw application_insights_id)

# Container App logs
az containerapp logs show \
  --name $(terraform output -raw container_app_name) \
  --resource-group $(terraform output -raw resource_group_name) \
  --follow
```

## Disaster Recovery

- **Database Backups**: 7-day retention (configurable)
- **Point-in-time restore**: Enabled by default
- **Geo-redundant backups**: Optional (additional cost)

## Cleanup

To destroy all resources:

```bash
cd terraform/environments/prod
terraform destroy
```

⚠️ **Warning**: This will delete all data. Ensure backups are in place.

## Troubleshooting

### Common Issues

**Issue: Terraform state locked**
```bash
# Force unlock (use carefully)
terraform force-unlock <lock-id>
```

**Issue: Container app failing to start**
```bash
# Check logs
az containerapp logs show \
  --name <app-name> \
  --resource-group <rg-name> \
  --tail 100
```

**Issue: Database connection failed**
```bash
# Verify firewall rules
az postgres flexible-server firewall-rule list \
  --resource-group <rg-name> \
  --name <server-name>
```

## Support

For issues or questions:
1. Check Application Insights logs
2. Review Azure Resource Health
3. Contact the development team

## References

- [Azure Container Apps Documentation](https://docs.microsoft.com/en-us/azure/container-apps/)
- [Terraform Azure Provider](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)
- [SendGrid Azure Integration](https://docs.sendgrid.com/for-developers/partners/microsoft-azure)

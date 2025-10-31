# Go Volunteer Media - Terraform Infrastructure

This directory contains the Terraform infrastructure as code (IaC) for deploying the Go Volunteer Media application to Azure.

## Architecture

- **Azure Container Apps**: Serverless container hosting with auto-scaling
- **PostgreSQL Flexible Server**: Managed database (Burstable B1ms tier)
- **Azure Blob Storage**: Image upload storage
- **SendGrid**: Email service (SMTP)
- **GitHub Container Registry**: Container image storage (free)
- **Application Insights**: Monitoring and observability
- **Key Vault**: Secrets management

## Prerequisites

1. **Azure CLI**: Install from https://docs.microsoft.com/en-us/cli/azure/install-azure-cli
2. **Terraform**: Version 1.5.0 or later
3. **Azure Subscription**: Active subscription with appropriate permissions
4. **GitHub Account**: For GitHub Container Registry (GHCR)
5. **SendGrid Account**: Free tier from Azure Marketplace

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

### 1. Login to Azure

```bash
az login
az account set --subscription "Your-Subscription-ID"
```

### 2. Create Storage Account for Terraform State

```bash
# Set variables
RESOURCE_GROUP="rg-terraform-state"
STORAGE_ACCOUNT="sttfstate$(openssl rand -hex 4)"
CONTAINER_NAME="tfstate"
LOCATION="eastus"

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create storage account
az storage account create \
  --resource-group $RESOURCE_GROUP \
  --name $STORAGE_ACCOUNT \
  --sku Standard_LRS \
  --encryption-services blob

# Create container
az storage container create \
  --name $CONTAINER_NAME \
  --account-name $STORAGE_ACCOUNT

echo "Storage Account: $STORAGE_ACCOUNT"
```

### 3. Configure Backend

Update `environments/prod/backend.tf` with your storage account name.

### 4. Create SendGrid Account

```bash
# Option 1: Via Azure Portal
# 1. Go to Azure Portal
# 2. Create Resource → Search "SendGrid"
# 3. Select Free tier (25k emails/month)
# 4. Complete setup

# Option 2: Via Azure CLI (after account setup)
# Get API key from SendGrid portal: https://app.sendgrid.com
```

### 5. Initialize Terraform

```bash
cd terraform/environments/prod
terraform init
```

## Deployment

### Development Environment

```bash
cd terraform/environments/dev
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

### Production Environment

```bash
cd terraform/environments/prod
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

## Configuration

### Required Variables

Create a `terraform.tfvars` file in the environment directory:

```hcl
# terraform/environments/prod/terraform.tfvars
project_name = "volunteer-media"
environment  = "prod"
location     = "eastus"

# Container configuration
container_image = "ghcr.io/networkengineer-cloud/go-volunteer-media:latest"
container_registry_server = "ghcr.io"
container_registry_username = "networkengineer-cloud"
# container_registry_password - set via environment variable

# Database configuration
db_admin_username = "vmadmin"
# db_admin_password - set via environment variable

# SendGrid configuration
sendgrid_api_key = "SG.xxx"  # Get from SendGrid portal
smtp_from_email  = "noreply@yourvolunteerorg.com"

# GitHub OAuth (for container registry)
github_token = "ghp_xxx"  # Set via environment variable

# Admin email for alerts
admin_email = "admin@yourvolunteerorg.com"

# Cost optimization
monthly_budget = 25
```

### Sensitive Variables via Environment Variables

```bash
export TF_VAR_db_admin_password="YourSecurePassword123!"
export TF_VAR_container_registry_password="ghp_your_github_token"
export TF_VAR_github_token="ghp_your_github_token"
export TF_VAR_jwt_secret="your-super-secure-jwt-secret-min-32-chars"
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
- SendGrid: Free tier (25k emails/month)
- **Total: ~$16-20/month**

## Security Best Practices

1. **Never commit sensitive values** - Use environment variables or Azure Key Vault
2. **Use managed identities** - Avoid storing credentials
3. **Enable encryption** - All data encrypted at rest and in transit
4. **Network isolation** - Private endpoints for database
5. **RBAC** - Least privilege access
6. **Monitoring** - Application Insights for all resources
7. **Backup** - Automated database backups enabled

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

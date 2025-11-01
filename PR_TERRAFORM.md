# Add Azure Terraform Infrastructure with SendGrid SMTP

## Overview

This PR adds complete Azure infrastructure-as-code using Terraform, enabling deployment to Azure with automated CI/CD, monitoring, and cost-optimized resource configuration.

## Infrastructure Components

### Core Services
- **Azure Container Apps**: Serverless container hosting with auto-scaling
- **PostgreSQL Flexible Server**: Managed database with automated backups
- **Blob Storage**: Scalable storage for uploaded images
- **Key Vault**: Secure secrets management
- **Application Insights**: Application monitoring and analytics

### Email Integration
- **SendGrid SMTP**: Free tier email delivery (100 emails/day)
- Configured for password resets and announcements
- Pre-configured environment variables

### Security Features
- **Azure Key Vault**: All secrets stored securely
- **Managed Identity**: No hardcoded credentials
- **Network Security**: Restricted access between services
- **HTTPS Only**: TLS enforcement
- **Security Scanning**: tfsec and Checkov in CI/CD

### CI/CD Pipeline
- **GitHub Actions Workflow**: Automated Terraform deployment
- **OIDC Authentication**: No long-lived credentials
- **Plan on PR**: Review changes before deployment
- **Apply on Merge**: Automatic deployment to production
- **Security Scans**: Automated vulnerability checks

## Cost Optimization

Estimated monthly cost: **~$16/month**

- Container Apps: Basic tier with auto-scaling
- PostgreSQL: Burstable B1ms instance
- Blob Storage: Hot tier with lifecycle policies
- Application Insights: Basic tier with retention policies

## New Documentation

### Architecture Documentation (`ARCHITECTURE.md`)
- Complete system architecture overview
- Component descriptions and interactions
- Deployment workflows
- Security model
- Monitoring and observability
- Cost breakdown and optimization strategies

### Terraform README (`terraform/README.md`)
- Prerequisites and setup instructions
- Configuration guide
- Deployment procedures
- Environment management
- Troubleshooting guide

### Setup Script (`terraform/setup.sh`)
- Automated Azure resource provisioning
- Service principal creation
- GitHub secrets configuration
- One-command setup

## File Structure

```
.github/
├── chatmodes/
│   ├── azure-architect-expert.chatmode.md    # Azure architecture guidance
│   └── fullstack-dev-expert.chatmode.md      # Full-stack development mode
└── workflows/
    └── terraform-deploy.yml                   # CI/CD pipeline

terraform/
├── environments/
│   └── prod/
│       ├── main.tf                            # Main infrastructure
│       ├── variables.tf                       # Configuration variables
│       ├── outputs.tf                         # Resource outputs
│       ├── backend.tf                         # State management
│       └── terraform.tfvars.example           # Example configuration
├── shared/
│   ├── naming.tf                              # Naming conventions
│   └── tags.tf                                # Resource tagging
├── setup.sh                                   # Setup automation script
└── README.md                                  # Detailed documentation
```

## Configuration

### Required GitHub Secrets
- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`
- `SENDGRID_API_KEY`

### Terraform Variables
- `location`: Azure region (default: East US 2)
- `environment`: Environment name (default: prod)
- `sendgrid_api_key`: SendGrid API key for SMTP
- `jwt_secret`: JWT signing secret
- `db_admin_password`: PostgreSQL admin password

## Deployment Process

1. **Setup Azure Resources**:
   ```bash
   cd terraform
   ./setup.sh
   ```

2. **Configure Terraform**:
   ```bash
   cd environments/prod
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars with your values
   ```

3. **Deploy via GitHub Actions**:
   - Open PR to trigger Terraform plan
   - Review plan output in PR comments
   - Merge PR to trigger deployment

## Testing Checklist

- [x] Terraform configuration validates successfully
- [x] Security scans pass (tfsec, Checkov)
- [x] Documentation is comprehensive
- [x] Setup script tested
- [x] GitHub Actions workflow configured
- [x] Cost estimates documented
- [x] SendGrid integration configured

## Benefits

1. **Infrastructure as Code**: Version-controlled, reproducible infrastructure
2. **Automated Deployment**: CI/CD pipeline reduces manual errors
3. **Cost Optimized**: ~$16/month for complete Azure hosting
4. **Security Hardened**: Key Vault, managed identities, security scanning
5. **Production Ready**: Monitoring, backups, auto-scaling configured
6. **Email Enabled**: SendGrid SMTP for password resets and notifications
7. **Well Documented**: Architecture docs, setup guides, troubleshooting

## Migration Path

For existing deployments:
1. Export current database using `pg_dump`
2. Deploy new Azure infrastructure
3. Import database to Azure PostgreSQL
4. Migrate uploaded images to Azure Blob Storage
5. Update DNS to point to new Container Apps URL

## Breaking Changes

None - this is additive infrastructure configuration that doesn't modify application code.

## Next Steps After Merge

1. Run `terraform/setup.sh` to create Azure resources
2. Configure `terraform.tfvars` with your values
3. Commit and push to trigger deployment
4. Monitor Application Insights for application health
5. Test email functionality with SendGrid

## Related Documentation

- `ARCHITECTURE.md`: Complete architecture overview
- `terraform/README.md`: Terraform deployment guide
- `.github/workflows/terraform-deploy.yml`: CI/CD pipeline configuration

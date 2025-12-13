# Dev Environment Setup

Quick guide to deploy development environment to Azure via HCP Terraform.

## Prerequisites

- Azure CLI: `az login`
- HCP Terraform account at https://app.terraform.io
- Resend account at https://resend.com (free tier: 3,000 emails/month)

## 1. Run Azure AD Setup

```bash
cd terraform/environments/dev
./setup-azure-ad-dev.sh
```

Save the `AZURE_CLIENT_ID_DEV` value from the output.

## 2. Configure HCP Terraform Workspace

**Workspace:** `Networkengineer/volunteer-app-dev`  
**URL:** https://app.terraform.io/app/Networkengineer/workspaces/volunteer-app-dev/variables

### Environment Variables (Azure auth only):
```
ARM_CLIENT_ID = <from-script>
ARM_TENANT_ID = 0d9bb6ac-d0d5-4194-8d89-4a0cc14d77f4
ARM_SUBSCRIPTION_ID = bd4191b8-87ee-4d65-917b-ddd3eddba8af
ARM_USE_OIDC = true
```

**Note:** No Terraform variables needed - all secrets passed via GitHub Actions

## 3. Add GitHub Secrets

**Repository:** https://github.com/networkengineer-cloud/go-volunteer-media/settings/secrets/actions

All application secrets go here (passed to Terraform by GitHub Actions):

```bash
# Azure authentication
AZURE_CLIENT_ID_DEV = <from setup script>

# Application secrets (dev-specific)
DEV_RESEND_API_KEY = <from resend.com>
DEV_JWT_SECRET = <generate with openssl>
DEV_CONTAINER_IMAGE = ghcr.io/networkengineer-cloud/go-volunteer-media:develop
```

Quick add with GitHub CLI:
```bash
gh secret set AZURE_CLIENT_ID_DEV --body "<paste-value>"
gh secret set DEV_RESEND_API_KEY --body "<paste-value>"
gh secret set DEV_JWT_SECRET --body "$(openssl rand -base64 32)"
gh secret set DEV_CONTAINER_IMAGE --body "ghcr.io/networkengineer-cloud/go-volunteer-media:develop"
```

## 4. Create GitHub Environment

1. Go to: https://github.com/networkengineer-cloud/go-volunteer-media/settings/environments
2. Click "New environment"
3. Name: `development`
4. Save

## 5. Deploy

### Via GitHub Actions:
1. Go to: https://github.com/networkengineer-cloud/go-volunteer-media/actions
2. Click "Terraform Deploy - Development"
3. Click "Run workflow"
4. Action: `plan` (first time to review)
5. Review output, then run again with action: `apply`

### Or push to develop branch:
```bash
git checkout -b develop
git push -u origin develop
```

## Configuration Summary

| Setting | Value |
|---------|-------|
| HCP Org | Networkengineer |
| Workspace | volunteer-app-dev |
| Region | centralus |
| Budget | $20/month with alerts |
| Auto-pause | DB after 60 min idle |
| Scale to zero | Yes |

## Troubleshooting

**Authentication failed:** Re-run `./setup-azure-ad-dev.sh`

**Database timeout:** Wait 2 minutes for auto-resume from pause

**Container not starting:** Check logs:
```bash
az containerapp logs show \
  --name ca-volunteer-media-dev \
  --resource-group rg-volunteer-media-dev \
  --tail 50
```

**Destroy to save costs:**
```bash
# Via GitHub Actions: Run workflow → Action: destroy
# Or locally:
terraform destroy
```

## Cost Estimate

- Container Apps: $2-5/month (scale to zero)
- PostgreSQL: $5-8/month (auto-pause)
- Storage: $1-2/month
- **Total: ~$10-15/month**

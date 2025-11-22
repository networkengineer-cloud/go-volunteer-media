# GitHub Secrets Checklist

This document lists all required GitHub secrets for CI/CD workflows.

## 📍 How to Add Secrets

1. Go to your GitHub repository
2. Click **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add each secret from the lists below

---

## 🔐 Required Secrets - Both Environments

### Azure Authentication (OIDC Federated Credentials)

| Secret Name | Description | How to Get |
|-------------|-------------|------------|
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID | Run: `az account show --query id -o tsv` |
| `AZURE_TENANT_ID` | Azure AD tenant ID | Run: `az account show --query tenantId -o tsv` |

### HCP Terraform

| Secret Name | Description | How to Get |
|-------------|-------------|------------|
| `TF_API_TOKEN` | HCP Terraform API token | Create at: https://app.terraform.io/app/settings/tokens |

### Testing

| Secret Name | Description | Example Value |
|-------------|-------------|---------------|
| `TEST_JWT_SECRET` | JWT secret for tests (optional, has default) | `L5WTt6D+6R55YfKzwqPRAEX5bR0bkNo4i58jYKL0wsk=` |

---

## 🏭 Production Environment Secrets

### Azure Authentication

| Secret Name | Description | How to Get |
|-------------|-------------|------------|
| `AZURE_CLIENT_ID` | Production Azure AD app client ID | From `terraform/environments/prod/setup-azure-ad.sh` output |

### Application Configuration

| Secret Name | Description | How to Get | Notes |
|-------------|-------------|------------|-------|
| `RESEND_API_KEY` | Resend SMTP API key | Create at: https://resend.com/api-keys | Replace SendGrid |
| `JWT_SECRET` | Production JWT signing key | Generate: `openssl rand -base64 32` | Min 32 chars |
| `OWNER_EMAIL` | Administrator email address | Your production admin email | For alerts |
| `CONTAINER_IMAGE` | Production container image URL | `ghcr.io/networkengineer-cloud/go-volunteer-media:latest` | Auto-built |

### Claude Integration (Optional)

| Secret Name | Description | How to Get |
|-------------|-------------|------------|
| `CLAUDE_CODE_OAUTH_TOKEN` | Claude Code API token | Create at: https://console.anthropic.com/ |

---

## 🧪 Development Environment Secrets

### Azure Authentication

| Secret Name | Description | How to Get |
|-------------|-------------|------------|
| `AZURE_CLIENT_ID_DEV` | Dev Azure AD app client ID | From `terraform/environments/dev/setup-azure-ad-dev.sh` output |

### Application Configuration

| Secret Name | Description | How to Get | Notes |
|-------------|-------------|------------|-------|
| `DEV_RESEND_API_KEY` | Dev Resend SMTP API key | Create at: https://resend.com/api-keys | Use test mode |
| `DEV_JWT_SECRET` | Dev JWT signing key | Generate: `openssl rand -base64 32` | Different from prod |
| `DEV_CONTAINER_IMAGE` | Dev container image URL | `ghcr.io/networkengineer-cloud/go-volunteer-media:latest-dev` | Auto-built |

---

## 🚀 Setup Sequence

### Initial Setup (One Time)

1. **Azure Authentication Setup**
   ```bash
   # Production
   cd terraform/environments/prod
   ./setup-azure-ad.sh
   # Save the AZURE_CLIENT_ID output
   
   # Development
   cd terraform/environments/dev
   ./setup-azure-ad-dev.sh
   # Save the AZURE_CLIENT_ID_DEV output
   ```

2. **Add Azure Secrets to GitHub**
   - `AZURE_SUBSCRIPTION_ID`
   - `AZURE_TENANT_ID`
   - `AZURE_CLIENT_ID` (from prod setup)
   - `AZURE_CLIENT_ID_DEV` (from dev setup)

3. **Create HCP Terraform Token**
   - Go to: https://app.terraform.io/app/settings/tokens
   - Create new token
   - Add as `TF_API_TOKEN` secret

4. **Generate Application Secrets**
   ```bash
   # Production JWT secret
   openssl rand -base64 32
   # Add as JWT_SECRET
   
   # Development JWT secret
   openssl rand -base64 32
   # Add as DEV_JWT_SECRET
   ```

5. **Setup Resend API Keys**
   - Go to: https://resend.com/api-keys
   - Create production API key → Add as `RESEND_API_KEY`
   - Create development API key (test mode) → Add as `DEV_RESEND_API_KEY`

6. **Add Owner Email**
   - Add your admin email as `OWNER_EMAIL`

### Container Images (Automatic)

Container images are built automatically by `.github/workflows/build-image.yml`:
- **Production**: Push to `main` → builds `ghcr.io/networkengineer-cloud/go-volunteer-media:latest`
- **Development**: Push to `develop` → builds `ghcr.io/networkengineer-cloud/go-volunteer-media:latest-dev`

**Important**: After first image build, add the image URLs as secrets:
- `CONTAINER_IMAGE` (production)
- `DEV_CONTAINER_IMAGE` (development)

---

## ✅ Verification Checklist

### Production Environment
- [ ] `AZURE_SUBSCRIPTION_ID` added
- [ ] `AZURE_TENANT_ID` added
- [ ] `AZURE_CLIENT_ID` added (from setup-azure-ad.sh)
- [ ] `TF_API_TOKEN` added
- [ ] `RESEND_API_KEY` added
- [ ] `JWT_SECRET` added (32+ chars)
- [ ] `OWNER_EMAIL` added
- [ ] `CONTAINER_IMAGE` added (after first build)
- [ ] GitHub Environment `production` created
- [ ] Production Azure AD app has federated credentials for HCP Terraform
- [ ] Production Azure AD app has federated credentials for GitHub Actions

### Development Environment
- [ ] `AZURE_CLIENT_ID_DEV` added (from setup-azure-ad-dev.sh)
- [ ] `DEV_RESEND_API_KEY` added
- [ ] `DEV_JWT_SECRET` added (32+ chars)
- [ ] `DEV_CONTAINER_IMAGE` added (after first build)
- [ ] GitHub Environment `development` created
- [ ] Dev Azure AD app has federated credentials for HCP Terraform
- [ ] Dev Azure AD app has federated credentials for GitHub Actions

### Optional
- [ ] `TEST_JWT_SECRET` added (optional, has default)
- [ ] `CLAUDE_CODE_OAUTH_TOKEN` added (for Claude code reviews)

---

## 🌍 GitHub Environments Setup

### Create Production Environment
1. Go to **Settings** → **Environments**
2. Click **New environment**
3. Name: `production`
4. Add protection rules:
   - ✅ Required reviewers (select team members)
   - ✅ Wait timer: 5 minutes (optional)
   - ✅ Deployment branches: `main` only

### Create Development Environment
1. Go to **Settings** → **Environments**
2. Click **New environment**
3. Name: `development`
4. Add protection rules (optional):
   - ✅ Deployment branches: `develop` only

---

## 🔍 Testing Your Setup

### Test Development Deployment
```bash
# Create and switch to develop branch
git checkout -b develop
git push -u origin develop

# Trigger manual deployment
# Go to: Actions → Terraform Deploy - Development → Run workflow
```

### Test Production Deployment
```bash
# Merge to main triggers automatic deployment
# Or use manual trigger:
# Go to: Actions → Terraform Deployment → Run workflow
```

### Verify Container Images
```bash
# Check images are being built
# Go to: Actions → Build Container Image

# View published images
# Go to: github.com/networkengineer-cloud/go-volunteer-media/pkgs/container/go-volunteer-media
```

---

## 🆘 Troubleshooting

### "Secret not found" Errors
- Ensure secret name matches exactly (case-sensitive)
- Check secret is added at repository level, not environment level
- Verify you're in the correct repository

### "OIDC token validation failed"
- Verify Azure AD app has federated credentials
- Check subject identifier matches:
  - HCP Terraform: `organization:Networkengineer:project:volunteer-app:workspace:volunteer-app-dev:run_phase:*`
  - GitHub Actions: `repo:networkengineer-cloud/go-volunteer-media:environment:development`

### "Terraform authentication failed"
- Verify `TF_API_TOKEN` is correct
- Check token has not expired
- Ensure organization name is `Networkengineer`

### Container Image Not Found
- Wait for first image build to complete
- Check Actions tab for build-image.yml workflow
- Verify GHCR packages are public or have correct permissions

---

## 📚 Reference Documentation

- [Azure OIDC Setup Guide](../terraform/FEDERATED_CREDENTIALS.md)
- [HCP Terraform Token Setup](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens)
- [Resend API Documentation](https://resend.com/docs/introduction)
- [GitHub Secrets Documentation](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [GitHub Environments](https://docs.github.com/en/actions/deployment/targeting-different-environments/using-environments-for-deployment)

---

*Last updated: November 22, 2025*

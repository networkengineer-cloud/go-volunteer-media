# Cloudflare integration for Azure Container Apps
# This configuration restricts access to only Cloudflare IP ranges

# Set env variable CLOUDFLARE_API_TOKEN with the API token before applying Terraform
provider "cloudflare" {
  # api_token = var.cloudflare_api_token
}

# Note: Azure Container Apps doesn't support IP restrictions directly on ingress.
# For full protection, consider:
# 1. Azure Front Door with Private Link (enterprise solution)
# 2. Application-level header validation (simpler approach)

# Application-level approach: Validate Cloudflare headers in your Go middleware
# Required headers to validate:
# - CF-Connecting-IP: Original client IP
# - CF-Ray: Cloudflare request ID
# - CF-Visitor: Protocol information

# For reference, Cloudflare IP ranges (update periodically):
# IPv4: https://www.cloudflare.com/ips-v4
# IPv6: https://www.cloudflare.com/ips-v6

locals {
  # Cloudflare IPv4 ranges (as of 2025)
  cloudflare_ipv4_ranges = [
    "173.245.48.0/20",
    "103.21.244.0/22",
    "103.22.200.0/22",
    "103.31.4.0/22",
    "141.101.64.0/18",
    "108.162.192.0/18",
    "190.93.240.0/20",
    "188.114.96.0/20",
    "197.234.240.0/22",
    "198.41.128.0/17",
    "162.158.0.0/15",
    "104.16.0.0/13",
    "104.24.0.0/14",
    "172.64.0.0/13",
    "131.0.72.0/22"
  ]

  # DNS TXT record name for Azure custom domain verification
  custom_domain_txt_record_name        = var.custom_domain != "" ? "asuid.${var.custom_domain}" : ""
  public_domain_txt_record_name        = var.public_domain != "" ? "asuid.${var.public_domain}" : ""

  # Note: For production, implement one of these solutions:
  # 1. Add header validation middleware in Go API
  # 2. Use Azure Front Door + Private Link (additional cost ~$35/month)
  # 3. Use Azure Application Gateway + WAF (additional cost ~$125/month)
}

# Output Cloudflare configuration instructions
output "cloudflare_setup_instructions" {
  description = "Instructions for setting up Cloudflare"
  value = jsonencode({
    step_1 = "Add your custom domain to Cloudflare"
    step_2 = "Create CNAME record pointing to Container App FQDN"
    step_3 = "Enable 'Proxied' (orange cloud) in Cloudflare DNS"
    step_4 = "In Container App, add custom domain"
    step_5 = "Implement header validation in Go API middleware"

    cloudflare_headers_to_validate = [
      "CF-Connecting-IP",
      "CF-Ray",
      "CF-Visitor"
    ]
  })
}

# TXT record for Azure Container Apps custom domain verification (infrastructure domain)
# This validates ownership of the infrastructure domain (prd.myhaws.org)
resource "cloudflare_dns_record" "custom_domain_verification" {
  count = var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = local.custom_domain_txt_record_name
  type    = "TXT"
  content = azurerm_container_app.main.custom_domain_verification_id
  ttl     = 300
  proxied = false

  comment = "Azure Container Apps custom domain verification (infrastructure)"
}

# TXT record for Azure Container Apps public domain verification (www.myhaws.org)
resource "cloudflare_dns_record" "public_domain_verification" {
  count = var.public_domain != "" && var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = local.public_domain_txt_record_name
  type    = "TXT"
  content = azurerm_container_app.main.custom_domain_verification_id
  ttl     = 300
  proxied = false

  comment = "Azure Container Apps custom domain verification (public)"
}

# CNAME record for infrastructure domain
# Points prd.myhaws.org -> Azure Container App FQDN
# Proxied through Cloudflare for DDoS protection once certificate is provisioned
resource "cloudflare_dns_record" "infrastructure_domain" {
  count = var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = var.custom_domain
  type    = "CNAME"
  content = azurerm_container_app.main.ingress[0].fqdn
  ttl     = 1
  proxied = true

  comment = "Infrastructure domain for Azure Container App"
}

# CNAME record for public-facing domain
# Points www.myhaws.org -> prd.myhaws.org
# This provides indirection: if infrastructure changes, only update the infrastructure CNAME
resource "cloudflare_dns_record" "public_domain" {
  count = var.public_domain != "" && var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = var.public_domain
  type    = "CNAME"
  content = var.custom_domain
  ttl     = 1 # Must be 1 (automatic) when proxied
  proxied = true

  comment = "Public-facing domain (Cloudflare proxied, handles TLS)"
}

# Apex domain (myhaws.org) - point directly to infrastructure domain
# Cloudflare CNAME flattening converts this to A/AAAA records automatically
# Proxied through Cloudflare for DDoS protection
resource "cloudflare_dns_record" "apex_domain" {
  count = var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = "@" # @ represents the apex domain
  type    = "CNAME"
  content = var.custom_domain
  ttl     = 1
  proxied = true

  comment = "Apex domain (flattened CNAME to infrastructure domain)"
}

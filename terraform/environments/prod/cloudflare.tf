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
  custom_domain_txt_record_name = var.custom_domain != "" ? "asuid.${var.custom_domain}" : ""
  
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
    
    security_note = "For enterprise-grade protection, consider Azure Front Door with Private Link endpoint"
  })
}

# TXT record for Azure Container Apps custom domain verification
resource "cloudflare_dns_record" "custom_domain_verification" {
  count = var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = local.custom_domain_txt_record_name
  type    = "TXT"
  content = azurerm_container_app.main.custom_domain_verification_id
  ttl     = 300
  proxied = false

  comment = "Azure Container Apps custom domain verification"
}

# CNAME record for application hostname
resource "cloudflare_dns_record" "domain" {
  count = var.custom_domain != "" && var.cloudflare_zone_id != "" ? 1 : 0

  zone_id = var.cloudflare_zone_id
  name    = var.custom_domain
  type    = "CNAME"
  content = azurerm_container_app.main.latest_revision_fqdn
  ttl     = 1
  proxied = false

  comment = "CNAME record for Azure Container App custom domain"
}

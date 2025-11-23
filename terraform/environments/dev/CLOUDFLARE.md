# Cloudflare Integration for Azure Container Apps

This setup restricts your application to only accept traffic from Cloudflare, providing DDoS protection, WAF, and performance benefits.

## Architecture

```
Internet → Cloudflare (Proxy) → Azure Container App → Backend
           ↑ Only this path allowed
```

## Implementation Method

Since Azure Container Apps doesn't support IP allow-listing on ingress directly, we use **application-level protection** via Go middleware that:

1. **Validates source IP** is from Cloudflare IP ranges
2. **Checks Cloudflare headers** (CF-Ray, CF-Connecting-IP)
3. **Extracts real client IP** from CF-Connecting-IP header
4. **Adds country and device info** to request context

## Setup Steps

### 1. Add Custom Domain to Cloudflare

1. Sign up at [Cloudflare](https://dash.cloudflare.com)
2. Add your domain (e.g., `volunteermedia.app`)
3. Update nameservers at your registrar to Cloudflare's

### 2. Create DNS Record

In Cloudflare DNS:

```
Type: CNAME
Name: dev (or your subdomain)
Target: <container-app-fqdn> (from terraform output)
Proxy status: Proxied (orange cloud) ✅
TTL: Auto
```

Example:
```
CNAME  dev  →  ca-volunteer-media-dev.kindocean-abc123.centralus.azurecontainerapps.io
```

### 3. Configure Azure Container App Custom Domain

```bash
# Get Container App FQDN
cd terraform/environments/dev
terraform output container_app_fqdn

# Add custom domain via Azure Portal or CLI
az containerapp hostname add \
  --name ca-volunteer-media-dev \
  --resource-group rg-volunteer-media-dev \
  --hostname dev.volunteermedia.app
```

### 4. Enable Middleware in Your App

In `cmd/api/main.go`:

```go
package main

import (
    "github.com/gin-gonic/gin"
    "your-app/internal/middleware"
)

func main() {
    router := gin.Default()
    
    // Apply Cloudflare protection globally
    router.Use(middleware.CloudflareOnly())        // Block non-Cloudflare IPs
    router.Use(middleware.CloudflareRealIP())      // Extract real client IP
    router.Use(middleware.CloudflareHeaders())     // Add CF headers to context
    
    // Your routes here
    router.GET("/api/health", healthHandler)
    
    router.Run(":8080")
}
```

**For development** (allow local testing):

```go
// Allow local IPs in development
if os.Getenv("ENVIRONMENT") == "development" {
    router.Use(middleware.DevelopmentBypass(true))
}
router.Use(middleware.CloudflareOnly())
```

## Security Features

### What This Protects Against

✅ **DDoS attacks** - Cloudflare absorbs attacks before reaching your app  
✅ **Direct IP access** - App only responds to Cloudflare IPs  
✅ **Bot attacks** - Cloudflare Bot Management  
✅ **SQL injection** - Cloudflare WAF  
✅ **XSS attacks** - Cloudflare WAF  
✅ **Layer 7 attacks** - Application-level protection  

### What You Get from Cloudflare

- **Free Tier Includes:**
  - Unlimited DDoS mitigation
  - Global CDN (300+ locations)
  - Free SSL/TLS certificates
  - Basic WAF rules
  - Analytics and logs
  - Rate limiting (coming soon to free tier)

- **Pro Tier ($20/month):**
  - Advanced WAF rules
  - Image optimization
  - Mobile optimization
  - Page Rules (20 rules)
  - Priority support

## Cloudflare Headers Available

The middleware extracts these headers for your use:

```go
// In your handlers, access via context:
cfRay := c.GetString("cf_ray")              // Unique request ID
realIP := c.GetString("cf_connecting_ip")   // Real client IP
country := c.GetString("cf_country")        // Client country code
deviceType := c.GetString("cf_device_type") // mobile/desktop/tablet
```

## Verification

### Test Cloudflare Protection

```bash
# This should SUCCEED (proxied through Cloudflare)
curl https://dev.volunteermedia.app/api/health

# This should FAIL (direct to Azure - bypassing Cloudflare)
curl https://ca-volunteer-media-dev.kindocean-abc123.centralus.azurecontainerapps.io/api/health
# Response: {"error":"Access denied - requests must come through Cloudflare"}
```

### Check Headers

```bash
curl -I https://dev.volunteermedia.app/api/health

# Look for Cloudflare headers:
# CF-Ray: 8c9a1b2c3d4e5f6g-DFW
# CF-Cache-Status: DYNAMIC
# Server: cloudflare
```

## Cloudflare Dashboard Configuration

### Recommended Settings

**SSL/TLS → Overview:**
- Encryption mode: **Full (strict)** ✅
  - Ensures end-to-end encryption
  - Cloudflare validates Azure's certificate

**SSL/TLS → Edge Certificates:**
- Always Use HTTPS: **On** ✅
- Minimum TLS Version: **TLS 1.2** ✅
- Opportunistic Encryption: **On**
- TLS 1.3: **On** ✅

**Speed → Optimization:**
- Auto Minify: **JavaScript, CSS, HTML** ✅
- Brotli: **On** ✅
- Early Hints: **On**
- HTTP/2 to Origin: **On**
- HTTP/3 (with QUIC): **On**

**Security → WAF:**
- Cloudflare Managed Ruleset: **On** ✅
- OWASP Core Ruleset: **On** ✅
- Cloudflare OWASP Core Ruleset: **On** ✅

**Security → Settings:**
- Security Level: **Medium** (adjust as needed)
- Challenge Passage: **30 minutes**
- Browser Integrity Check: **On** ✅

### Rate Limiting Rules (Pro+)

Create rate limiting rules to prevent abuse:

```yaml
# API rate limit
Rule: API Protection
Expression: (http.request.uri.path contains "/api/")
Characteristics: IP Address
Requests: 100
Period: 60 seconds
Action: Block
```

## Cost Comparison

| Solution | Monthly Cost | Features |
|----------|--------------|----------|
| **Cloudflare Free + App Middleware** | $0 | DDoS protection, basic WAF, CDN |
| Cloudflare Pro + App Middleware | $20 | Advanced WAF, image optimization |
| Azure Front Door + Private Link | ~$35 | Native Azure integration, private networking |
| Azure Application Gateway + WAF | ~$125 | Full Azure WAF, more control |

**Recommendation:** Start with **Cloudflare Free + App Middleware** (implemented here). Upgrade to Pro ($20/mo) if you need advanced WAF rules or image optimization.

## Monitoring

### Application Insights Queries

Track requests by country:

```kusto
requests
| extend Country = customDimensions.cf_country
| summarize count() by Country
| order by count_ desc
```

Track Cloudflare Ray IDs for debugging:

```kusto
requests
| extend CFRay = customDimensions.cf_ray
| where isnotempty(CFRay)
| project timestamp, url, resultCode, CFRay
```

### Cloudflare Analytics

Access in Cloudflare Dashboard:
- **Traffic** → See bandwidth, requests, caching
- **Security** → View threats blocked
- **Performance** → CDN performance metrics

## Troubleshooting

### "Access denied" error

**Problem:** App returns 403 Forbidden  
**Cause:** Request not coming through Cloudflare  
**Solution:** Ensure DNS is proxied (orange cloud) and you're using custom domain

### Headers missing in development

**Problem:** CF-Ray header is empty locally  
**Cause:** Running in debug mode  
**Solution:** Middleware automatically bypasses in `gin.DebugMode`

### Can't access app at all

**Problem:** All requests blocked  
**Cause:** Cloudflare IP ranges outdated  
**Solution:** Update IP ranges in `internal/middleware/cloudflare.go` from https://www.cloudflare.com/ips-v4

## Maintenance

**Update Cloudflare IP ranges quarterly:**

```bash
# Fetch latest ranges
curl https://www.cloudflare.com/ips-v4 > cloudflare-ips-v4.txt
curl https://www.cloudflare.com/ips-v6 > cloudflare-ips-v6.txt

# Update internal/middleware/cloudflare.go with new ranges
```

## Alternative: Enterprise Solution

For maximum security, consider **Azure Front Door + Private Link**:

```hcl
# terraform/environments/dev/frontdoor.tf
resource "azurerm_cdn_frontdoor_profile" "main" {
  name                = "afd-${var.project_name}-${var.environment}"
  resource_group_name = azurerm_resource_group.main.name
  sku_name            = "Standard_AzureFrontDoor"
}

# Make Container App private (no public ingress)
resource "azurerm_container_app" "main" {
  ingress {
    external_enabled = false  # Private only
    # ... rest of config
  }
}

# Connect Front Door via Private Link
# Then use Cloudflare → Azure Front Door → Private Container App
```

**Cost:** ~$35/month additional  
**Benefits:** True private networking, no public IP exposure, defense in depth

---

**Current Implementation:** Application-level Cloudflare validation (zero additional cost, excellent security)

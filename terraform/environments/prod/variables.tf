# Input variables for the production environment
# Override these values in terraform.tfvars

variable "project_name" {
  type        = string
  description = "Project name used in resource naming"
  default     = "volunteer-media"

  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.project_name))
    error_message = "Project name must contain only lowercase letters, numbers, and hyphens."
  }
}

variable "environment" {
  type        = string
  description = "Environment name (dev, staging, prod)"
  default     = "prod"

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}

variable "location" {
  type        = string
  description = "Azure region for resources"
  default     = "centralus"
}

variable "owner_email" {
  type        = string
  description = "Email address of the resource owner"
}

variable "cost_center" {
  type        = string
  description = "Cost center for billing"
  default     = "volunteer-operations"
}

variable "additional_tags" {
  type        = map(string)
  description = "Additional tags to apply to resources"
  default     = {}
}

# Container App Configuration
variable "container_image" {
  type        = string
  description = "Container image to deploy (e.g., ghcr.io/user/volunteer-media:latest)"
}

variable "container_registry_url" {
  type        = string
  description = "Container registry URL"
  default     = "ghcr.io"
}

variable "container_cpu" {
  type        = number
  description = "CPU cores for container (0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0)"
  default     = 0.5

  validation {
    condition     = contains([0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0], var.container_cpu)
    error_message = "CPU must be one of: 0.25, 0.5, 0.75, 1.0, 1.25, 1.5, 1.75, 2.0"
  }
}

variable "container_memory" {
  type        = string
  description = "Memory for container (0.5Gi, 1Gi, 1.5Gi, 2Gi, 3Gi, 4Gi)"
  default     = "1Gi"

  validation {
    condition     = contains(["0.5Gi", "1Gi", "1.5Gi", "2Gi", "3Gi", "4Gi"], var.container_memory)
    error_message = "Memory must be one of: 0.5Gi, 1Gi, 1.5Gi, 2Gi, 3Gi, 4Gi"
  }
}

variable "min_replicas" {
  type        = number
  description = "Minimum number of container replicas"
  default     = 1

  validation {
    condition     = var.min_replicas >= 0 && var.min_replicas <= 30
    error_message = "Min replicas must be between 0 and 30."
  }
}

variable "max_replicas" {
  type        = number
  description = "Maximum number of container replicas"
  default     = 3

  validation {
    condition     = var.max_replicas >= 1 && var.max_replicas <= 30
    error_message = "Max replicas must be between 1 and 30."
  }
}

# Database Configuration
variable "db_admin_username" {
  type        = string
  description = "PostgreSQL admin username"
  default     = "pgadmin"

  validation {
    condition     = can(regex("^[a-zA-Z][a-zA-Z0-9_]{2,62}$", var.db_admin_username))
    error_message = "Username must start with a letter and be 3-63 characters."
  }
}

variable "db_sku_name" {
  type        = string
  description = "PostgreSQL SKU (B_Standard_B1ms recommended for low cost)"
  default     = "B_Standard_B1ms"
}

variable "db_storage_mb" {
  type        = number
  description = "PostgreSQL storage in MB"
  default     = 32768 # 32 GB

  validation {
    condition     = var.db_storage_mb >= 32768 && var.db_storage_mb <= 16777216
    error_message = "Storage must be between 32 GB and 16 TB."
  }
}

variable "db_backup_retention_days" {
  type        = number
  description = "Number of days to retain backups (14 recommended for production)"
  default     = 14

  validation {
    condition     = var.db_backup_retention_days >= 7 && var.db_backup_retention_days <= 35
    error_message = "Backup retention must be between 7 and 35 days."
  }
}

variable "db_high_availability_enabled" {
  type        = bool
  description = "Enable high availability for PostgreSQL"
  default     = false # Disabled for cost savings
}

variable "db_auto_grow_enabled" {
  type        = bool
  description = "Enable auto-grow for PostgreSQL storage"
  default     = true
}

# Storage Configuration
variable "storage_account_tier" {
  type        = string
  description = "Storage account tier (Standard or Premium)"
  default     = "Standard"

  validation {
    condition     = contains(["Standard", "Premium"], var.storage_account_tier)
    error_message = "Storage tier must be Standard or Premium."
  }
}

variable "storage_replication_type" {
  type        = string
  description = "Storage replication type (ZRS recommended for production reliability)"
  default     = "ZRS" # Zone-redundant storage for better durability

  validation {
    condition     = contains(["LRS", "GRS", "RAGRS", "ZRS", "GZRS", "RAGZRS"], var.storage_replication_type)
    error_message = "Invalid replication type."
  }
}

# Email Configuration (Resend SMTP)
variable "resend_api_key" {
  type        = string
  description = "Resend API key for SMTP authentication"
  sensitive   = true
}

variable "resend_from_email" {
  type        = string
  description = "Default 'from' email address for Resend"

  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.resend_from_email))
    error_message = "Must be a valid email address."
  }
}

variable "resend_from_name" {
  type        = string
  description = "Display name for the email sender"
  default     = "MyHAWS"
}

# Monitoring Configuration
variable "log_retention_days" {
  type        = number
  description = "Number of days to retain logs"
  default     = 30

  validation {
    condition     = var.log_retention_days >= 30 && var.log_retention_days <= 730
    error_message = "Log retention must be between 30 and 730 days."
  }
}

variable "monthly_budget_amount" {
  type        = number
  description = "Monthly budget alert threshold in USD"
  default     = 50

  validation {
    condition     = var.monthly_budget_amount > 0
    error_message = "Budget amount must be positive."
  }
}

variable "budget_alert_emails" {
  type        = list(string)
  description = "Email addresses to notify for budget alerts"
  default     = []
}

# Application Configuration
variable "jwt_secret" {
  type        = string
  description = "JWT secret for authentication"
  sensitive   = true
}

variable "allowed_origins" {
  type        = list(string)
  description = "Allowed CORS origins (should be explicit domains in production)"
  default = [
    "https://www.myhaws.org",
    "https://myhaws.org"
  ]

  validation {
    condition     = !contains(var.allowed_origins, "*") || var.environment != "prod"
    error_message = "Wildcard CORS (*) is not allowed in production. Specify explicit domains."
  }
}

# Frontend Configuration
variable "frontend_url" {
  type        = string
  description = "Frontend URL for password reset links and CORS. Must be accessible by end users receiving emails. Used in password reset email links and API CORS configuration."
  default     = "https://www.myhaws.org"

  validation {
    condition     = can(regex("^https?://", var.frontend_url))
    error_message = "Frontend URL must start with http:// or https://"
  }
}

# Custom Domain Configuration
variable "custom_domain" {
  type        = string
  description = "Infrastructure domain for Azure Container App binding (e.g., prd.myhaws.org for prod, dev.myhaws.org for dev). This is the target of the public-facing CNAME."
  default     = "prd.myhaws.org"

  validation {
    condition     = var.custom_domain == "" || can(regex("^([a-z0-9]([a-z0-9-]*[a-z0-9])?.)+[a-z]{2,}$", var.custom_domain))
    error_message = "Custom domain must be a valid domain name or empty string."
  }
}

variable "public_domain" {
  type        = string
  description = "Public-facing domain for users (e.g., www.myhaws.org). CNAME points this to custom_domain. Used in frontend_url and allowed_origins."
  default     = "www.myhaws.org"

  validation {
    condition     = var.public_domain == "" || can(regex("^([a-z0-9]([a-z0-9-]*[a-z0-9])?.)+[a-z]{2,}$", var.public_domain))
    error_message = "Public domain must be a valid domain name or empty string."
  }
}

variable "custom_domain_certificate_id" {
  type        = string
  description = "Resource ID of a Container App Environment Certificate to bind via SNI. Leave empty to use Azure Managed Certificate."
  default     = ""
}

variable "cloudflare_zone_id" {
  type        = string
  description = "Cloudflare Zone ID for the domain (e.g., myhaws.org)."
  default     = ""

  validation {
    condition     = var.custom_domain == "" || var.cloudflare_zone_id != ""
    error_message = "cloudflare_zone_id must be set when custom_domain is provided."
  }
}

# variable "cloudflare_api_token" {
#   type        = string
#   description = "Cloudflare API token with DNS edit permissions for the zone."
#   sensitive   = true
#   default     = ""

#   validation {
#     condition     = var.custom_domain == "" || var.cloudflare_api_token != ""
#     error_message = "cloudflare_api_token must be set when custom_domain is provided."
#   }
# }

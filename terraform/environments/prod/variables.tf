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
  default     = "eastus"
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
  default     = 32768  # 32 GB
  
  validation {
    condition     = var.db_storage_mb >= 32768 && var.db_storage_mb <= 16777216
    error_message = "Storage must be between 32 GB and 16 TB."
  }
}

variable "db_backup_retention_days" {
  type        = number
  description = "Number of days to retain backups"
  default     = 7
  
  validation {
    condition     = var.db_backup_retention_days >= 7 && var.db_backup_retention_days <= 35
    error_message = "Backup retention must be between 7 and 35 days."
  }
}

variable "db_high_availability_enabled" {
  type        = bool
  description = "Enable high availability for PostgreSQL"
  default     = false  # Disabled for cost savings
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
  description = "Storage replication type (LRS, GRS, RAGRS, ZRS)"
  default     = "LRS"  # Locally redundant for cost savings
  
  validation {
    condition     = contains(["LRS", "GRS", "RAGRS", "ZRS", "GZRS", "RAGZRS"], var.storage_replication_type)
    error_message = "Invalid replication type."
  }
}

# SendGrid Configuration
variable "sendgrid_api_key" {
  type        = string
  description = "SendGrid API key for SMTP authentication"
  sensitive   = true
}

variable "sendgrid_from_email" {
  type        = string
  description = "Default 'from' email address for SendGrid"
  
  validation {
    condition     = can(regex("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.sendgrid_from_email))
    error_message = "Must be a valid email address."
  }
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
  description = "Allowed CORS origins"
  default     = ["*"]
}

variable "github_container_registry_username" {
  type        = string
  description = "GitHub username for GHCR authentication"
  default     = ""
}

variable "github_container_registry_password" {
  type        = string
  description = "GitHub Personal Access Token for GHCR"
  sensitive   = true
  default     = ""
}

# Frontend Configuration
variable "frontend_url" {
  type        = string
  description = "Frontend URL for password reset links and CORS. Must be accessible by end users receiving emails. Used in password reset email links and API CORS configuration."
  default     = "https://myhaws.org"
  
  validation {
    condition     = can(regex("^https?://", var.frontend_url))
    error_message = "Frontend URL must start with http:// or https://"
  }
}

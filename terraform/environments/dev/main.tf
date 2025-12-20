# Main Terraform configuration for Volunteer Media Platform - Development Environment
# Optimized for cost savings with auto-scaling and auto-pause features

# Get current Azure client configuration
data "azurerm_client_config" "current" {}

# Generate a random password for PostgreSQL admin
# Note: Password is stored in Terraform state. State is encrypted in HCP Terraform Cloud.
# For production, consider external secret generation and rotation strategy.
resource "random_password" "db_password" {
  length  = 32
  special = true
  
  lifecycle {
    ignore_changes = [
      length,
      special
    ]
  }
}

# Resource Group
resource "azurerm_resource_group" "main" {
  name     = "rg-${var.project_name}-${var.environment}"
  location = var.location
  
  tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "Terraform"
      Repository  = "go-volunteer-media"
      Owner       = var.owner_email
      CostCenter  = var.cost_center
      AutoShutdown = "enabled"
    },
    var.additional_tags
  )
}

# Log Analytics Workspace (required for Container Apps and monitoring)
resource "azurerm_log_analytics_workspace" "main" {
  name                = "log-${var.project_name}-${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = var.log_retention_days
  
  tags = azurerm_resource_group.main.tags
}

# Application Insights for monitoring
resource "azurerm_application_insights" "main" {
  name                = "appi-${var.project_name}-${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  workspace_id        = azurerm_log_analytics_workspace.main.id
  application_type    = "web"
  sampling_percentage = 100  # Full sampling for dev debugging
  
  tags = azurerm_resource_group.main.tags
}

# Key Vault for storing secrets
resource "azurerm_key_vault" "main" {
  name                       = "kv-${var.project_name}-${var.environment}"
  location                   = azurerm_resource_group.main.location
  resource_group_name        = azurerm_resource_group.main.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  
  # Security settings (more relaxed for dev)
  purge_protection_enabled   = false  # Allow immediate deletion in dev
  soft_delete_retention_days = 7      # Minimum retention for dev
  
  # Network access (open for dev ease of use)
  network_acls {
    default_action = "Allow"
    bypass         = "AzureServices"
  }
  
  tags = azurerm_resource_group.main.tags
}

# Grant Terraform access to Key Vault
resource "azurerm_key_vault_access_policy" "terraform" {
  key_vault_id = azurerm_key_vault.main.id
  tenant_id    = data.azurerm_client_config.current.tenant_id
  object_id    = data.azurerm_client_config.current.object_id
  
  secret_permissions = [
    "Get", "List", "Set", "Delete", "Purge", "Recover"
  ]
}

# Store database password in Key Vault
resource "azurerm_key_vault_secret" "db_password" {
  name         = "postgresql-admin-password"
  value        = random_password.db_password.result
  key_vault_id = azurerm_key_vault.main.id
  
  depends_on = [azurerm_key_vault_access_policy.terraform]
}

# Store Resend API key in Key Vault
resource "azurerm_key_vault_secret" "resend_api_key" {
  name         = "resend-api-key"
  value        = var.resend_api_key
  key_vault_id = azurerm_key_vault.main.id
  
  depends_on = [azurerm_key_vault_access_policy.terraform]
}

# Store JWT secret in Key Vault
resource "azurerm_key_vault_secret" "jwt_secret" {
  name         = "jwt-secret"
  value        = var.jwt_secret
  key_vault_id = azurerm_key_vault.main.id
  
  depends_on = [azurerm_key_vault_access_policy.terraform]
}

# PostgreSQL Flexible Server with auto-pause capability
resource "azurerm_postgresql_flexible_server" "main" {
  name                = "psql-${var.project_name}-${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  # Authentication
  administrator_login    = var.db_admin_username
  administrator_password = random_password.db_password.result
  
  # SKU Configuration (B_Standard_B1ms = Burstable tier with auto-pause)
  sku_name   = var.db_sku_name
  storage_mb = var.db_storage_mb
  version    = "15"  # PostgreSQL 15
  
  # Backup and HA
  backup_retention_days        = var.db_backup_retention_days
  geo_redundant_backup_enabled = false  # Disabled for cost savings
  auto_grow_enabled            = var.db_auto_grow_enabled
  
  # High availability disabled for dev (omit block entirely to disable)
  # high_availability {
  #   mode = "ZoneRedundant"  # Only valid values: "ZoneRedundant" or "SameZone"
  # }
  
  # Lifecycle protection (relaxed for dev)
  lifecycle {
    prevent_destroy = false  # Allow deletion in dev
    ignore_changes  = [
      zone  # Azure may assign zone automatically, ignore changes to prevent errors
    ]
  }
  
  tags = azurerm_resource_group.main.tags
}

# Note: PostgreSQL Flexible Server does not support auto-pause like Azure SQL Database.
# Cost savings achieved through:
# - Burstable B_Standard_B1ms SKU (pay only for compute used)
# - Storage auto-grow enabled
# - No high availability (single zone)
# - Minimal backup retention (7 days)

# PostgreSQL Firewall Rule - Allow Azure Services
resource "azurerm_postgresql_flexible_server_firewall_rule" "azure_services" {
  name             = "allow-azure-services"
  server_id        = azurerm_postgresql_flexible_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "0.0.0.0"
}

# PostgreSQL Database
resource "azurerm_postgresql_flexible_server_database" "main" {
  name      = "volunteermedia"
  server_id = azurerm_postgresql_flexible_server.main.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}

# Storage Account for image uploads
resource "azurerm_storage_account" "main" {
  name                = "st${replace(var.project_name, "-", "")}${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  account_tier             = var.storage_account_tier
  account_replication_type = var.storage_replication_type
  
  # Security settings
  min_tls_version                   = "TLS1_2"
  https_traffic_only_enabled        = true
  allow_nested_items_to_be_public   = false
  
  # Enable blob versioning for data protection
  blob_properties {
    versioning_enabled = true
    
    delete_retention_policy {
      days = 7
    }
    
    container_delete_retention_policy {
      days = 7
    }
  }
  
  tags = azurerm_resource_group.main.tags
}

# Blob container for animal images
resource "azurerm_storage_container" "uploads" {
  name                  = "uploads"
  storage_account_name  = azurerm_storage_account.main.name
  container_access_type = "private"
}

# Container App Environment
resource "azurerm_container_app_environment" "main" {
  name                       = "cae-${var.project_name}-${var.environment}"
  location                   = azurerm_resource_group.main.location
  resource_group_name        = azurerm_resource_group.main.name
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  
  tags = azurerm_resource_group.main.tags
}

# Container App
resource "azurerm_container_app" "main" {
  name                         = "ca-${var.project_name}-${var.environment}"
  container_app_environment_id = azurerm_container_app_environment.main.id
  resource_group_name          = azurerm_resource_group.main.name
  revision_mode                = "Single"
  
  # Container configuration
  template {
    min_replicas = 1  # Always keep minimum 1 replica running
    max_replicas = var.max_replicas
    
    # HTTP-based autoscaling
    http_scale_rule {
      name                = "http-scaler"
      concurrent_requests = "10"
    }
    
    container {
      name   = "volunteer-media-api"
      image  = var.container_image
      cpu    = var.container_cpu
      memory = var.container_memory
      
      # Environment variables
      env {
        name  = "GIN_MODE"
        value = "debug"  # Debug mode for dev
      }
      
      env {
        name  = "PORT"
        value = "8080"
      }
      
      # Environment identifier for logging and monitoring
      env {
        name  = "ENV"
        value = "development"
      }
      
      env {
        name  = "DB_HOST"
        value = azurerm_postgresql_flexible_server.main.fqdn
      }
      
      env {
        name  = "DB_PORT"
        value = "5432"
      }
      
      env {
        name  = "DB_USER"
        value = var.db_admin_username
      }
      
      env {
        name        = "DB_PASSWORD"
        secret_name = "db-password"
      }
      
      env {
        name  = "DB_NAME"
        value = azurerm_postgresql_flexible_server_database.main.name
      }
      
      env {
        name  = "DB_SSLMODE"
        value = "require"
      }
      
      # Database Logging (reduce verbosity)
      env {
        name  = "DB_LOG_LEVEL"
        value = "warn"
      }
      
      # Email Configuration (DISABLED for dev)
      env {
        name  = "EMAIL_ENABLED"
        value = "false"
      }
      
      env {
        name  = "EMAIL_PROVIDER"
        value = "resend"
      }
      
      # Frontend URL (for password reset links when email is enabled)
      env {
        name  = "FRONTEND_URL"
        value = var.frontend_url
      }
      
      # Resend SMTP Configuration (placeholder - email disabled)
      env {
        name  = "SMTP_HOST"
        value = "smtp.resend.com"
      }
      
      env {
        name  = "SMTP_PORT"
        value = "587"
      }
      
      env {
        name  = "SMTP_USER"
        value = "resend"
      }
      
      env {
        name        = "SMTP_PASS"
        secret_name = "resend-api-key"
      }
      
      env {
        name  = "SMTP_FROM"
        value = var.resend_from_email
      }
      
      # Azure Storage Configuration
      env {
        name  = "AZURE_STORAGE_ACCOUNT"
        value = azurerm_storage_account.main.name
      }
      
      env {
        name        = "AZURE_STORAGE_KEY"
        secret_name = "storage-account-key"
      }
      
      env {
        name  = "AZURE_STORAGE_CONTAINER"
        value = azurerm_storage_container.uploads.name
      }
      
      # JWT Secret
      env {
        name        = "JWT_SECRET"
        secret_name = "jwt-secret"
      }
      
      # CORS Configuration (more permissive for dev)
      env {
        name  = "ALLOWED_ORIGINS"
        value = join(",", var.allowed_origins)
      }
      
      # Application Insights
      env {
        name  = "APPLICATIONINSIGHTS_CONNECTION_STRING"
        value = azurerm_application_insights.main.connection_string
      }
    }
  }
  
  # Ingress configuration
  ingress {
    external_enabled = true
    target_port      = 8080
    
    traffic_weight {
      latest_revision = true
      percentage      = 100
    }
    
    # Allow HTTP/2 and HTTPS
    transport = "auto"
  }
  
  # Note: Custom domain with managed certificate must be added via Azure CLI after DNS validation
  # See outputs for step-by-step instructions
  
  # Secrets
  secret {
    name  = "db-password"
    value = random_password.db_password.result
  }
  
  secret {
    name  = "resend-api-key"
    value = var.resend_api_key
  }
  
  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }
  
  secret {
    name  = "storage-account-key"
    value = azurerm_storage_account.main.primary_access_key
  }
  
  # Note: No registry configuration needed - GHCR image is public
  
  # Managed Identity
  identity {
    type = "SystemAssigned"
  }
  
  tags = azurerm_resource_group.main.tags
}

# Custom Domain Configuration (only if custom domain is provided)
# Bind custom domain to Container App
resource "azurerm_container_app_custom_domain" "main" {
  count = var.custom_domain != "" ? 1 : 0
  
  name                  = var.custom_domain
  container_app_id      = azurerm_container_app.main.id
  certificate_binding_type = var.custom_domain_certificate_id != "" ? "SniEnabled" : "Disabled"
}

# Grant Container App managed identity access to Key Vault
resource "azurerm_key_vault_access_policy" "container_app" {
  key_vault_id = azurerm_key_vault.main.id
  tenant_id    = azurerm_container_app.main.identity[0].tenant_id
  object_id    = azurerm_container_app.main.identity[0].principal_id
  
  secret_permissions = [
    "Get", "List"
  ]
}

# Note: Role assignments require elevated permissions.
# For dev environment, we use storage account key directly in secrets.
# For production, use Managed Identity with pre-configured role assignments.
# Uncomment below if your service principal has Microsoft.Authorization/roleAssignments/write permission:
#
# resource "azurerm_role_assignment" "container_app_storage" {
#   principal_id         = azurerm_container_app.main.identity[0].principal_id
#   role_definition_name = "Storage Blob Data Contributor"
#   scope                = azurerm_storage_account.main.id
# }

# Diagnostic settings for Container App
# Note: Azure Container Apps only support metrics, not log categories
# Logs are automatically sent to the Container App Environment's Log Analytics workspace
resource "azurerm_monitor_diagnostic_setting" "container_app" {
  name                       = "diag-container-app"
  target_resource_id         = azurerm_container_app.main.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  
  metric {
    category = "AllMetrics"
    enabled  = true
  }
}

# Action Group for alerts
resource "azurerm_monitor_action_group" "main" {
  name                = "ag-${var.project_name}-${var.environment}"
  resource_group_name = azurerm_resource_group.main.name
  short_name          = "devalerts"
  
  email_receiver {
    name          = "admin"
    email_address = var.owner_email
  }
  
  tags = azurerm_resource_group.main.tags
}

# Alert: Application errors
resource "azurerm_monitor_metric_alert" "app_errors" {
  name                = "alert-app-errors-${var.project_name}-${var.environment}"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.main.id]
  description         = "Alert when application has high error rate"
  severity            = 2
  frequency           = "PT5M"
  window_size         = "PT15M"
  
  criteria {
    metric_namespace = "Microsoft.App/containerApps"
    metric_name      = "Replicas"
    aggregation      = "Average"
    operator         = "LessThan"
    threshold        = 1
  }
  
  action {
    action_group_id = azurerm_monitor_action_group.main.id
  }
  
  tags = azurerm_resource_group.main.tags
}

# Budget alert
resource "azurerm_consumption_budget_resource_group" "main" {
  name              = "budget-${var.project_name}-${var.environment}"
  resource_group_id = azurerm_resource_group.main.id
  
  amount     = var.monthly_budget_amount
  time_grain = "Monthly"
  
  time_period {
    start_date = formatdate("YYYY-MM-01'T'00:00:00Z", timestamp())
  }
  
  notification {
    enabled   = true
    threshold = 80
    operator  = "GreaterThan"
    
    contact_emails = concat(
      [var.owner_email],
      var.budget_alert_emails
    )
  }
  
  notification {
    enabled   = true
    threshold = 100
    operator  = "GreaterThan"
    
    contact_emails = concat(
      [var.owner_email],
      var.budget_alert_emails
    )
  }
  
  lifecycle {
    ignore_changes = [
      time_period  # Prevent recreation on every apply
    ]
  }
}

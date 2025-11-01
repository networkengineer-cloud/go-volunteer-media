# Main Terraform configuration for Volunteer Media Platform - Production Environment
# This configures all Azure resources needed to run the application

# Reference shared configuration
module "naming" {
  source = "../../shared"
}

# Get current Azure client configuration
data "azurerm_client_config" "current" {}

# Generate a random password for PostgreSQL admin
resource "random_password" "db_password" {
  length  = 32
  special = true
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
  
  tags = azurerm_resource_group.main.tags
}

# Key Vault for storing secrets
resource "azurerm_key_vault" "main" {
  name                       = "kv-${var.project_name}-${var.environment}"
  location                   = azurerm_resource_group.main.location
  resource_group_name        = azurerm_resource_group.main.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  
  # Security settings
  purge_protection_enabled   = true
  soft_delete_retention_days = 90
  
  # Allow Azure services to access Key Vault
  network_acls {
    default_action = "Allow"  # For initial setup; restrict after deployment
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

# Store SendGrid API key in Key Vault
resource "azurerm_key_vault_secret" "sendgrid_api_key" {
  name         = "sendgrid-api-key"
  value        = var.sendgrid_api_key
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

# PostgreSQL Flexible Server
resource "azurerm_postgresql_flexible_server" "main" {
  name                = "psql-${var.project_name}-${var.environment}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  
  # Authentication
  administrator_login    = var.db_admin_username
  administrator_password = random_password.db_password.result
  
  # SKU Configuration (B_Standard_B1ms = 1 vCore, 2 GB RAM)
  sku_name   = var.db_sku_name
  storage_mb = var.db_storage_mb
  version    = "15"  # PostgreSQL 15
  
  # Backup and HA
  backup_retention_days        = var.db_backup_retention_days
  geo_redundant_backup_enabled = false  # Disabled for cost savings
  
  high_availability {
    mode = var.db_high_availability_enabled ? "ZoneRedundant" : "Disabled"
  }
  
  # Lifecycle protection
  lifecycle {
    prevent_destroy = true  # Prevent accidental deletion
  }
  
  tags = azurerm_resource_group.main.tags
}

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
  name                 = "uploads"
  storage_account_id   = azurerm_storage_account.main.id
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
    min_replicas = var.min_replicas
    max_replicas = var.max_replicas
    
    container {
      name   = "volunteer-media-api"
      image  = var.container_image
      cpu    = var.container_cpu
      memory = var.container_memory
      
      # Environment variables
      env {
        name  = "GIN_MODE"
        value = "release"
      }
      
      env {
        name  = "PORT"
        value = "8080"
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
      
      # SendGrid SMTP Configuration
      env {
        name  = "SMTP_HOST"
        value = "smtp.sendgrid.net"
      }
      
      env {
        name  = "SMTP_PORT"
        value = "587"
      }
      
      env {
        name  = "SMTP_USER"
        value = "apikey"
      }
      
      env {
        name        = "SMTP_PASS"
        secret_name = "sendgrid-api-key"
      }
      
      env {
        name  = "SMTP_FROM"
        value = var.sendgrid_from_email
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
      
      # CORS Configuration
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
  
  # Secrets
  secret {
    name  = "db-password"
    value = random_password.db_password.result
  }
  
  secret {
    name  = "sendgrid-api-key"
    value = var.sendgrid_api_key
  }
  
  secret {
    name  = "jwt-secret"
    value = var.jwt_secret
  }
  
  secret {
    name  = "storage-account-key"
    value = azurerm_storage_account.main.primary_access_key
  }
  
  # Registry configuration for GHCR (if credentials provided)
  dynamic "registry" {
    for_each = var.github_container_registry_username != "" ? [1] : []
    
    content {
      server               = var.container_registry_url
      username             = var.github_container_registry_username
      password_secret_name = "ghcr-password"
    }
  }
  
  dynamic "secret" {
    for_each = var.github_container_registry_password != "" ? [1] : []
    
    content {
      name  = "ghcr-password"
      value = var.github_container_registry_password
    }
  }
  
  # Managed Identity
  identity {
    type = "SystemAssigned"
  }
  
  tags = azurerm_resource_group.main.tags
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

# Grant Container App managed identity access to Storage Account
resource "azurerm_role_assignment" "container_app_storage" {
  principal_id         = azurerm_container_app.main.identity[0].principal_id
  role_definition_name = "Storage Blob Data Contributor"
  scope                = azurerm_storage_account.main.id
}

# Diagnostic settings for Container App
resource "azurerm_monitor_diagnostic_setting" "container_app" {
  name                       = "diag-container-app"
  target_resource_id         = azurerm_container_app.main.id
  log_analytics_workspace_id = azurerm_log_analytics_workspace.main.id
  
  enabled_log {
    category = "ContainerAppConsoleLogs"
  }
  
  enabled_log {
    category = "ContainerAppSystemLogs"
  }
  
  metric {
    category = "AllMetrics"
  }
}

# Action Group for alerts
resource "azurerm_monitor_action_group" "main" {
  name                = "ag-${var.project_name}-${var.environment}"
  resource_group_name = azurerm_resource_group.main.name
  short_name          = "alerts"
  
  email_receiver {
    name          = "admin"
    email_address = var.owner_email
  }
  
  tags = azurerm_resource_group.main.tags
}

# Alert: High CPU usage
resource "azurerm_monitor_metric_alert" "cpu_high" {
  name                = "alert-cpu-high-${var.project_name}"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.main.id]
  description         = "Alert when CPU usage exceeds 80%"
  severity            = 2
  frequency           = "PT5M"
  window_size         = "PT15M"
  
  criteria {
    metric_namespace = "Microsoft.App/containerApps"
    metric_name      = "UsageNanoCores"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 800000000  # 80% of 1 core (in nanocores)
  }
  
  action {
    action_group_id = azurerm_monitor_action_group.main.id
  }
  
  tags = azurerm_resource_group.main.tags
}

# Alert: High memory usage
resource "azurerm_monitor_metric_alert" "memory_high" {
  name                = "alert-memory-high-${var.project_name}"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = [azurerm_container_app.main.id]
  description         = "Alert when memory usage exceeds 80%"
  severity            = 2
  frequency           = "PT5M"
  window_size         = "PT15M"
  
  criteria {
    metric_namespace = "Microsoft.App/containerApps"
    metric_name      = "WorkingSetBytes"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 838860800  # 80% of 1GB in bytes
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

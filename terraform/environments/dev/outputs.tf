# Output values for the development environment
# These values can be used for verification and debugging

output "resource_group_name" {
  description = "Name of the resource group"
  value       = azurerm_resource_group.main.name
}

output "resource_group_id" {
  description = "ID of the resource group"
  value       = azurerm_resource_group.main.id
}

output "container_app_fqdn" {
  description = "FQDN of the Container App"
  value       = azurerm_container_app.main.ingress[0].fqdn
}

output "container_app_url" {
  description = "Full URL of the Container App"
  value       = "https://${azurerm_container_app.main.ingress[0].fqdn}"
}

output "container_app_id" {
  description = "ID of the Container App"
  value       = azurerm_container_app.main.id
}

output "container_app_identity_principal_id" {
  description = "Principal ID of the Container App managed identity"
  value       = azurerm_container_app.main.identity[0].principal_id
}

output "postgresql_server_fqdn" {
  description = "FQDN of the PostgreSQL server"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "postgresql_server_id" {
  description = "ID of the PostgreSQL server"
  value       = azurerm_postgresql_flexible_server.main.id
}

output "postgresql_database_name" {
  description = "Name of the PostgreSQL database"
  value       = azurerm_postgresql_flexible_server_database.main.name
}

output "storage_account_name" {
  description = "Name of the storage account"
  value       = azurerm_storage_account.main.name
}

output "storage_account_id" {
  description = "ID of the storage account"
  value       = azurerm_storage_account.main.id
}

output "storage_account_primary_blob_endpoint" {
  description = "Primary blob endpoint of the storage account"
  value       = azurerm_storage_account.main.primary_blob_endpoint
}

output "storage_container_name" {
  description = "Name of the uploads storage container"
  value       = azurerm_storage_container.uploads.name
}

output "key_vault_id" {
  description = "ID of the Key Vault"
  value       = azurerm_key_vault.main.id
}

output "key_vault_uri" {
  description = "URI of the Key Vault"
  value       = azurerm_key_vault.main.vault_uri
}

output "log_analytics_workspace_id" {
  description = "ID of the Log Analytics workspace"
  value       = azurerm_log_analytics_workspace.main.id
}

output "application_insights_connection_string" {
  description = "Connection string for Application Insights"
  value       = azurerm_application_insights.main.connection_string
  sensitive   = true
}

output "application_insights_instrumentation_key" {
  description = "Instrumentation key for Application Insights"
  value       = azurerm_application_insights.main.instrumentation_key
  sensitive   = true
}

output "application_insights_app_id" {
  description = "App ID for Application Insights"
  value       = azurerm_application_insights.main.app_id
}

# Database connection information (for manual verification)
output "database_connection_info" {
  description = "Database connection information"
  value = {
    host     = azurerm_postgresql_flexible_server.main.fqdn
    port     = 5432
    database = azurerm_postgresql_flexible_server_database.main.name
    username = var.db_admin_username
    sslmode  = "require"
  }
}

# Resend configuration (for verification)
output "resend_configuration" {
  description = "Resend SMTP configuration"
  value = {
    smtp_host  = "smtp.resend.com"
    smtp_port  = 587
    smtp_user  = "resend"
    from_email = var.resend_from_email
  }
}

# Cost monitoring
output "monthly_budget_amount" {
  description = "Monthly budget alert threshold"
  value       = var.monthly_budget_amount
}

# Auto-pause configuration
output "database_auto_pause_config" {
  description = "Database auto-pause configuration"
  value = {
    enabled       = true
    delay_minutes = 60
    sku           = var.db_sku_name
  }
}

# Custom domain configuration
output "custom_domain_setup" {
  description = "Instructions for setting up custom domain with managed certificate"
  value = var.custom_domain != "" ? {
    step_1_dns = "Add CNAME record in your DNS provider:"
    cname_record = "${var.custom_domain} -> ${azurerm_container_app.main.ingress[0].fqdn}"
    
    step_2_verify = "Wait for DNS propagation (use: dig ${var.custom_domain})"
    
    step_3_add_domain = "Add custom domain with managed certificate using Azure CLI:"
    cli_command = "az containerapp hostname add --hostname ${var.custom_domain} --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_container_app.main.name} --location ${azurerm_resource_group.main.location}"
    
    step_4_bind_cert = "Bind managed certificate (Azure will auto-provision a free certificate):"
    bind_command = "az containerapp hostname bind --hostname ${var.custom_domain} --resource-group ${azurerm_resource_group.main.name} --name ${azurerm_container_app.main.name} --environment ${azurerm_container_app_environment.main.name} --validation-method CNAME"
    
    note = "Azure will automatically provision a free managed certificate after DNS validation completes"
  } : {
    message = "No custom domain configured. Set 'custom_domain' variable to enable."
  }
}

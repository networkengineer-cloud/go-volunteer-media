# Naming convention for Azure resources
# Follow Azure naming best practices: https://learn.microsoft.com/en-us/azure/cloud-adoption-framework/ready/azure-best-practices/resource-naming

locals {
  # Resource naming pattern: {resource-type}-{project}-{environment}-{region}
  
  naming = {
    resource_group       = "rg-${var.project_name}-${var.environment}"
    container_app_env    = "cae-${var.project_name}-${var.environment}"
    container_app        = "ca-${var.project_name}-${var.environment}"
    postgresql_server    = "psql-${var.project_name}-${var.environment}"
    storage_account      = "st${replace(var.project_name, "-", "")}${var.environment}"  # No hyphens allowed
    key_vault            = "kv-${var.project_name}-${var.environment}"
    log_analytics        = "log-${var.project_name}-${var.environment}"
    app_insights         = "appi-${var.project_name}-${var.environment}"
    action_group         = "ag-${var.project_name}-${var.environment}"
    budget               = "budget-${var.project_name}-${var.environment}"
  }
  
  # Location short codes for naming
  location_short = {
    eastus      = "eus"
    eastus2     = "eus2"
    westus      = "wus"
    westus2     = "wus2"
    centralus   = "cus"
    northcentralus = "ncus"
    southcentralus = "scus"
  }
  
  location_code = lookup(local.location_short, var.location, "unk")
}

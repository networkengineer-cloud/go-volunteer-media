# Common tags for all Azure resources
# Helps with cost tracking, compliance, and resource management

locals {
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
    Repository  = "go-volunteer-media"
    Owner       = var.owner_email
    CostCenter  = var.cost_center
  }
  
  # Merge common tags with any additional tags
  all_tags = merge(
    local.common_tags,
    var.additional_tags
  )
}

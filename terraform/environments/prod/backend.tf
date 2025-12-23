# Backend configuration for Terraform state storage
# Store state in HCP Terraform with federated credentials
# This file configures remote state management for the production environment

terraform {
  # Require Terraform 1.5 or later for cloud block with project support
  required_version = ">= 1.5.0, < 2.0.0"
  
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.117"  # Require 3.117+ for latest security patches
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.7"  # Require 3.7+ for latest features
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 5.0"  # Cloudflare provider for DNS TXT verification
    }
  }
  
  # HCP Terraform backend configuration with remote execution
  # Docs: https://developer.hashicorp.com/terraform/language/settings/terraform-cloud
  cloud {
    organization = "Networkengineer"  # HCP Terraform organization name
    
    workspaces {
      name    = "volunteer-app"  # Production workspace
      project = "HAWS"            # Project for workspace grouping
    }
  }
}

# Configure the Azure Provider
# Uses HCP Terraform Dynamic Provider Credentials (workload identity federation)
# Docs: https://developer.hashicorp.com/terraform/cloud-docs/dynamic-provider-credentials/azure-configuration
provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = false
      recover_soft_deleted_key_vaults = true
    }
    
    resource_group {
      prevent_deletion_if_contains_resources = true
    }
  }
  
  # HCP Terraform will automatically inject OIDC credentials via TFC_AZURE_* environment variables
  # Do NOT set: client_id, use_oidc, oidc_token, ARM_CLIENT_ID, ARM_USE_OIDC
  # Required HCP Terraform env vars: TFC_AZURE_PROVIDER_AUTH=true, TFC_AZURE_RUN_CLIENT_ID
  # Required provider args: subscription_id, tenant_id (set via ARM_SUBSCRIPTION_ID, ARM_TENANT_ID)
  use_cli = false  # Disable Azure CLI fallback for clearer error messages
}

# Backend configuration for Terraform state storage - Development Environment
# Store state in HCP Terraform with federated credentials
# This file configures remote state management for the development environment

terraform {
  required_version = ">= 1.5.0"
  
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.80"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
  
  # HCP Terraform backend configuration
  # Sign up at https://app.terraform.io and create an organization
  cloud {
    organization = "Networkengineer"  # HCP Terraform organization name
    
    workspaces {
      name = "volunteer-app-dev"  # Development workspace
    }
  }
}

# Configure the Azure Provider
# Uses federated credentials (OIDC) for authentication
provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = false
      recover_soft_deleted_key_vaults = true
    }
    
    resource_group {
      prevent_deletion_if_contains_resources = false  # More flexible for dev
    }
  }
  
  # Use OIDC authentication - no client secrets needed
  # Configured automatically when running in GitHub Actions or HCP Terraform
  use_oidc = true
}

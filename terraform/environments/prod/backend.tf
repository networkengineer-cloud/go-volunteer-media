# Backend configuration for Terraform state storage
# Store state in Azure Storage Account with state locking enabled
# This file configures remote state management for the production environment

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
  
  # Backend configuration for remote state
  # Create the storage account and container before running terraform init
  backend "azurerm" {
    resource_group_name  = "rg-terraform-state"
    storage_account_name = "sttfstatevolunteer"  # Must be globally unique
    container_name       = "tfstate"
    key                  = "volunteer-media/prod/terraform.tfstate"
    use_oidc            = true  # Use OIDC authentication for GitHub Actions
  }
}

# Configure the Azure Provider
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
  
  # Automatically uses OIDC when running in GitHub Actions
  use_oidc = true
}

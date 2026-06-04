#!/bin/bash
# Script to retrieve the database password from HCP Terraform state

set -e

echo "🔑 Retrieving Database Password from HCP Terraform"
echo "==================================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must run from project root directory"
    exit 1
fi

# Parse command line arguments
ENVIRONMENT="prod"

while [[ $# -gt 0 ]]; do
    case $1 in
        --env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --help|-h)
            echo ""
            echo "Usage: $0 [--env prod|dev]"
            echo ""
            echo "Options:"
            echo "  --env    Environment: prod or dev (default: prod)"
            echo ""
            echo "Requirements:"
            echo "  - terraform CLI installed and authenticated with HCP Terraform"
            echo "  - Run from project root directory"
            echo ""
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--env prod|dev]"
            exit 1
            ;;
    esac
done

# Validate environment
if [ "$ENVIRONMENT" != "prod" ] && [ "$ENVIRONMENT" != "dev" ]; then
    echo "❌ Error: Invalid environment '$ENVIRONMENT'. Must be 'prod' or 'dev'"
    exit 1
fi

TERRAFORM_DIR="terraform/environments/$ENVIRONMENT"

echo ""
echo "🔍 Environment: $ENVIRONMENT"
echo "📂 Terraform dir: $TERRAFORM_DIR"
echo ""

# Check if terraform is installed
if ! command -v terraform &> /dev/null; then
    echo "❌ Error: terraform CLI not found"
    echo "   Install from: https://developer.hashicorp.com/terraform/install"
    exit 1
fi

# Check if the terraform directory exists
if [ ! -d "$TERRAFORM_DIR" ]; then
    echo "❌ Error: Terraform directory not found: $TERRAFORM_DIR"
    exit 1
fi

cd "$TERRAFORM_DIR"

echo "📊 Fetching password from HCP Terraform state..."

DB_PASSWORD=$(terraform output -raw postgresql_admin_password 2>&1)

if [[ $DB_PASSWORD == *"Error"* ]] || [[ $DB_PASSWORD == *"error"* ]] || [ -z "$DB_PASSWORD" ]; then
    echo ""
    echo "❌ Failed to retrieve password automatically."
    echo ""
    echo "Possible causes:"
    echo "  - Not authenticated: run 'terraform login'"
    echo "  - State not initialized: run 'terraform init' in $TERRAFORM_DIR"
    echo "  - Output not present in this environment's Terraform config"
    echo ""
    echo "Manual retrieval:"
    if [ "$ENVIRONMENT" = "prod" ]; then
        echo "  https://app.terraform.io/app/Networkengineer/volunteer-app-prod"
    else
        echo "  https://app.terraform.io/app/Networkengineer/volunteer-app-dev"
    fi
    echo "  → States tab → latest state → search for 'postgresql_admin_password'"
    echo ""
    exit 1
fi

echo ""
echo "=============================="
echo "✅ DB Password Retrieved"
echo "=============================="
echo ""
echo "🔑 Password: $DB_PASSWORD"
echo ""
echo "⚠️  Keep this password secure. Do not share or commit it."
echo ""

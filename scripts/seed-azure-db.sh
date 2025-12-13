#!/bin/bash
# Script to seed the Azure PostgreSQL database from local machine

set -e

echo "🌱 Seeding Azure PostgreSQL Database"
echo "======================================"

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must run from project root directory"
    exit 1
fi

# Set Azure database connection details
export DB_HOST="psql-volunteer-media-dev.postgres.database.azure.com"
export DB_PORT="5432"
export DB_USER="pgadmin"
export DB_NAME="volunteermedia"
export DB_SSLMODE="require"

# Get password from HCP Terraform state (requires HCP Terraform CLI)
echo "📊 Fetching database password from HCP Terraform..."
cd terraform/environments/dev

# Try to get password from terraform output
if command -v terraform &> /dev/null; then
    DB_PASSWORD=$(terraform output -raw postgresql_server_fqdn 2>&1 | head -1)
    
    # If that doesn't work, we need to guide the user
    if [[ $DB_PASSWORD == *"Error"* ]] || [[ -z "$DB_PASSWORD" ]]; then
        echo ""
        echo "⚠️  Cannot automatically retrieve password."
        echo ""
        echo "Please get the password manually:"
        echo "  1. Go to: https://app.terraform.io/app/Networkengineer/volunteer-app-dev"
        echo "  2. Click on 'States' tab"
        echo "  3. View the latest state"
        echo "  4. Search for 'postgresql-admin-password' or 'db_password'"
        echo ""
        echo "Then run this command with the password:"
        echo "  DB_PASSWORD='<password>' bash scripts/seed-azure-db.sh"
        echo ""
        
        if [ -z "$DB_PASSWORD" ]; then
            echo "❌ Error: DB_PASSWORD environment variable not set"
            echo "   Set it and run again: DB_PASSWORD='your-password' bash scripts/seed-azure-db.sh"
            exit 1
        fi
    fi
fi

cd ../../..

# Check if DB_PASSWORD is set
if [ -z "$DB_PASSWORD" ]; then
    echo "❌ Error: DB_PASSWORD not set"
    echo "   Set it manually: DB_PASSWORD='your-password' bash scripts/seed-azure-db.sh"
    exit 1
fi

export DB_PASSWORD

echo "✅ Database connection configured"
echo "   Host: $DB_HOST"
echo "   Database: $DB_NAME"
echo "   User: $DB_USER"
echo ""

# Run seed command
echo "🚀 Running seed command..."
echo ""

go run cmd/seed/main.go "$@"

echo ""
echo "✅ Seeding complete!"
echo ""
echo "🌐 Test the app: https://ca-volunteer-media-dev.bluebush-4a7df924.centralus.azurecontainerapps.io"
echo ""

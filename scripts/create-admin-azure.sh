#!/bin/bash
# Script to create a production admin account by running a job in Azure Container Apps
# This bypasses firewall issues by running inside Azure's network

set -e

echo "👤 Creating Production Admin Account via Azure Container Apps"
echo "=============================================================="

# Parse command line arguments
ENVIRONMENT="prod"
ADMIN_USERNAME="terry"
ADMIN_EMAIL=""
FORCE_RESET=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --email)
            ADMIN_EMAIL="$2"
            shift 2
            ;;
        --env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --force)
            FORCE_RESET="--force"
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 --email admin@example.com [--env prod|dev] [--force]"
            exit 1
            ;;
    esac
done

# Validate email is provided
if [ -z "$ADMIN_EMAIL" ]; then
    echo "❌ Error: Email address is required"
    echo ""
    echo "Usage: $0 --email admin@example.com [--env prod|dev] [--force]"
    echo ""
    echo "Options:"
    echo "  --email    Admin email address (required)"
    echo "  --env      Environment: prod or dev (default: prod)"
    echo "  --force    Reset password if user already exists"
    echo ""
    exit 1
fi

# Set resource names based on environment
if [ "$ENVIRONMENT" = "prod" ]; then
    RESOURCE_GROUP="rg-volunteer-media-prod"
    CONTAINER_APP="ca-volunteer-media-prod"
elif [ "$ENVIRONMENT" = "dev" ]; then
    RESOURCE_GROUP="rg-volunteer-media-dev"
    CONTAINER_APP="ca-volunteer-media-dev"
else
    echo "❌ Error: Invalid environment '$ENVIRONMENT'. Must be 'prod' or 'dev'"
    exit 1
fi

echo ""
echo "🔍 Environment: $ENVIRONMENT"
echo "📦 Resource Group: $RESOURCE_GROUP"
echo "🚀 Container App: $CONTAINER_APP"
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo "❌ Error: Azure CLI not found"
    echo "   Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check if logged in
if ! az account show &> /dev/null; then
    echo "❌ Error: Not logged in to Azure CLI"
    echo "   Run: az login"
    exit 1
fi

echo "🔐 Generating secure password..."
ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d "=+/" | cut -c1-16)

echo "✅ Password generated"
echo ""

# Execute command in the container app
echo "🚀 Creating admin account in Azure..."
echo "   (This runs inside the container app to bypass firewall)"
echo ""

# Build the command to run inside the container
CREATE_CMD="
export ADMIN_USERNAME='$ADMIN_USERNAME'
export ADMIN_EMAIL='$ADMIN_EMAIL'
export ADMIN_PASSWORD='$ADMIN_PASSWORD'

cat > /tmp/create-admin.go << 'GOEOF'
package main

import (
	\"fmt\"
	\"log\"
	\"os\"

	\"golang.org/x/crypto/bcrypt\"
	\"gorm.io/driver/postgres\"
	\"gorm.io/gorm\"
)

type User struct {
	ID                        uint   \\\`gorm:\"primaryKey\"\\\`
	Username                  string \\\`gorm:\"uniqueIndex;not null\"\\\`
	Email                     string \\\`gorm:\"uniqueIndex;not null\"\\\`
	Password                  string \\\`gorm:\"not null\"\\\`
	IsAdmin                   bool   \\\`gorm:\"default:false\"\\\`
	EmailNotificationsEnabled bool   \\\`gorm:\"default:false\"\\\`
	PhoneNumber               string
	HideEmail                 bool \\\`gorm:\"default:false\"\\\`
	HidePhoneNumber           bool \\\`gorm:\"default:false\"\\\`
}

func main() {
	username := os.Getenv(\"ADMIN_USERNAME\")
	email := os.Getenv(\"ADMIN_EMAIL\")
	password := os.Getenv(\"ADMIN_PASSWORD\")

	if username == \"\" || email == \"\" || password == \"\" {
		log.Fatal(\"ADMIN_USERNAME, ADMIN_EMAIL, and ADMIN_PASSWORD must be set\")
	}

	dsn := fmt.Sprintf(\"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s\",
		os.Getenv(\"DB_HOST\"),
		os.Getenv(\"DB_USER\"),
		os.Getenv(\"DB_PASSWORD\"),
		os.Getenv(\"DB_NAME\"),
		os.Getenv(\"DB_PORT\"),
		os.Getenv(\"DB_SSLMODE\"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf(\"Failed to connect to database: %v\", err)
	}

	var existingUser User
	result := db.Where(\"username = ? OR email = ?\", username, email).First(&existingUser)
	
	if result.Error == nil {
		log.Printf(\"User '%s' already exists. Use --force to reset password.\", username)
		os.Exit(1)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf(\"Failed to hash password: %v\", err)
	}

	newUser := User{
		Username:                  username,
		Email:                     email,
		Password:                  string(hashedPassword),
		IsAdmin:                   true,
		EmailNotificationsEnabled: false,
		PhoneNumber:               \"\",
		HideEmail:                 false,
		HidePhoneNumber:           false,
	}

	if err := db.Create(&newUser).Error; err != nil {
		log.Fatalf(\"Failed to create user: %v\", err)
	}

	fmt.Printf(\"Admin user '%s' created successfully\", username)
}
GOEOF

cd /tmp
go run create-admin.go
"

# Execute in the running container
az containerapp exec \
  --name "$CONTAINER_APP" \
  --resource-group "$RESOURCE_GROUP" \
  --command "/bin/sh" \
  --stdin <<< "$CREATE_CMD"

EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo ""
    echo "======================================"
    echo "✅ Admin Account Created Successfully!"
    echo "======================================"
    echo ""
    echo "👤 Username: $ADMIN_USERNAME"
    echo "📧 Email:    $ADMIN_EMAIL"
    echo "🔑 Password: $ADMIN_PASSWORD"
    echo ""
    echo "⚠️  IMPORTANT: Save these credentials securely!"
    echo "   Store the password in a password manager."
    echo ""
    if [ "$ENVIRONMENT" = "prod" ]; then
        echo "🌐 Login at: https://myhaws.org/login"
    else
        echo "🌐 Login at: https://ca-volunteer-media-dev.bluebush-4a7df924.centralus.azurecontainerapps.io/login"
    fi
    echo ""
else
    echo ""
    echo "❌ Failed to create admin account"
    echo "   Check the error messages above"
    echo ""
    exit 1
fi

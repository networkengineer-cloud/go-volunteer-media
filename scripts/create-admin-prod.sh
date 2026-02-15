#!/bin/bash
# Script to create a production admin account in Azure PostgreSQL database
# Creates a single admin user named "Terry" with a secure generated password

set -e

echo "👤 Creating Production Admin Account"
echo "====================================="

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo "❌ Error: Must run from project root directory"
    exit 1
fi

# Parse command line arguments
ENVIRONMENT="prod"
ADMIN_USERNAME="terry"
ADMIN_EMAIL=""
FORCE_RESET=false

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
            FORCE_RESET=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--email admin@example.com] [--env prod|dev] [--force]"
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

# Set database connection details based on environment
if [ "$ENVIRONMENT" = "prod" ]; then
    export DB_HOST="psql-volunteer-media-prod.postgres.database.azure.com"
elif [ "$ENVIRONMENT" = "dev" ]; then
    export DB_HOST="psql-volunteer-media-dev.postgres.database.azure.com"
else
    echo "❌ Error: Invalid environment '$ENVIRONMENT'. Must be 'prod' or 'dev'"
    exit 1
fi

export DB_PORT="5432"
export DB_USER="pgadmin"
export DB_NAME="volunteermedia"
export DB_SSLMODE="require"

# Generate a secure random password (16 characters with special chars)
ADMIN_PASSWORD=$(openssl rand -base64 12 | tr -d "=+/" | cut -c1-16)

echo ""
echo "🔐 Generated secure password for admin account"
echo ""

# Get database password from environment or prompt user
if [ -z "$DB_PASSWORD" ]; then
    echo "⚠️  Database password required to connect"
    echo ""
    echo "Please get the PostgreSQL admin password:"
    if [ "$ENVIRONMENT" = "prod" ]; then
        echo "  1. Go to: https://app.terraform.io/app/Networkengineer/workspaces/volunteer-app"
    else
        echo "  1. Go to: https://app.terraform.io/app/Networkengineer/workspaces/volunteer-app-dev"
    fi
    echo "  2. Click on 'Variables' tab"
    echo "  3. Find the 'postgresql_admin_password' output value"
    echo "  Or retrieve from Azure Key Vault secret: 'postgresql-admin-password'"
    echo ""
    echo "Then run this command with the password:"
    echo "  DB_PASSWORD='<db-password>' bash scripts/create-admin-prod.sh --email $ADMIN_EMAIL --env $ENVIRONMENT"
    echo ""
    exit 1
fi

export DB_PASSWORD

echo "✅ Database connection configured"
echo "   Environment: $ENVIRONMENT"
echo "   Host: $DB_HOST"
echo "   Database: $DB_NAME"
echo "   User: $DB_USER"
echo ""

# Create temporary Go program to create the admin user
cat > /tmp/create-admin.go << 'EOF'
package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID                        uint   `gorm:"primaryKey"`
	Username                  string `gorm:"uniqueIndex;not null"`
	Email                     string `gorm:"uniqueIndex;not null"`
	Password                  string `gorm:"not null"`
	IsAdmin                   bool   `gorm:"default:false"`
	EmailNotificationsEnabled bool   `gorm:"default:false"`
	PhoneNumber               string
	HideEmail                 bool `gorm:"default:false"`
	HidePhoneNumber           bool `gorm:"default:false"`
}

func main() {
	username := os.Getenv("ADMIN_USERNAME")
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	forceReset := os.Getenv("FORCE_RESET") == "true"

	if username == "" || email == "" || password == "" {
		log.Fatal("ADMIN_USERNAME, ADMIN_EMAIL, and ADMIN_PASSWORD must be set")
	}

	// Build connection string
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Check if user already exists
	var existingUser User
	result := db.Where("username = ? OR email = ?", username, email).First(&existingUser)
	
	if result.Error == nil {
		// User exists
		if forceReset {
			// Reset password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash password: %v", err)
			}

			existingUser.Password = string(hashedPassword)
			existingUser.IsAdmin = true
			existingUser.Email = email

			if err := db.Save(&existingUser).Error; err != nil {
				log.Fatalf("Failed to update user: %v", err)
			}

			fmt.Printf("✅ Admin user '%s' updated successfully (password reset)\n", username)
		} else {
			fmt.Printf("⚠️  User '%s' already exists. Use --force to reset password.\n", username)
			os.Exit(1)
		}
	} else {
		// Create new user
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}

		newUser := User{
			Username:                  username,
			Email:                     email,
			Password:                  string(hashedPassword),
			IsAdmin:                   true,
			EmailNotificationsEnabled: false,
			PhoneNumber:               "",
			HideEmail:                 false,
			HidePhoneNumber:           false,
		}

		if err := db.Create(&newUser).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}

		fmt.Printf("✅ Admin user '%s' created successfully\n", username)
	}
}
EOF

# Export admin credentials for the Go program
export ADMIN_USERNAME
export ADMIN_EMAIL
export ADMIN_PASSWORD
export FORCE_RESET

# Run the Go program with required dependencies
echo "🚀 Creating admin account..."
echo ""

cd /tmp
go mod init create-admin 2>/dev/null || true
go get golang.org/x/crypto/bcrypt@latest 2>/dev/null || true
go get gorm.io/driver/postgres@latest 2>/dev/null || true
go get gorm.io/gorm@latest 2>/dev/null || true
go run create-admin.go

# Cleanup
rm -f create-admin.go

cd - > /dev/null

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
echo "   Store the password in a password manager and delete from terminal history."
echo ""
if [ "$ENVIRONMENT" = "prod" ]; then
    echo "🌐 Login at: https://myhaws.org/login"
else
    echo "🌐 Login at: https://ca-volunteer-media-dev.bluebush-4a7df924.centralus.azurecontainerapps.io/login"
fi
echo ""

package main

import (
"fmt"
"log"

"github.com/joho/godotenv"
"github.com/networkengineer-cloud/go-volunteer-media/internal/auth"
"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
"github.com/networkengineer-cloud/go-volunteer-media/internal/models"
)

func main() {
// Load environment
_ = godotenv.Load()

// Initialize database
db, err := database.Initialize()
if err != nil {
log.Fatal("Failed to initialize database:", err)
}

// Run migrations
if err := database.RunMigrations(db); err != nil {
log.Fatal("Failed to run migrations:", err)
}

// Check if admin user exists
var count int64
db.Model(&models.User{}).Where("username = ?", "testadmin").Count(&count)
if count > 0 {
fmt.Println("Admin user already exists")
} else {
// Create admin user
adminPassword, _ := auth.HashPassword("password123")
admin := models.User{
Username: "testadmin",
Email:    "admin@test.com",
Password: adminPassword,
IsAdmin:  true,
}
if err := db.Create(&admin).Error; err != nil {
log.Fatal("Failed to create admin user:", err)
}
fmt.Println("Created admin user: testadmin")
}

// Check if test user exists
db.Model(&models.User{}).Where("username = ?", "testuser").Count(&count)
if count > 0 {
fmt.Println("Test user already exists")
} else {
// Create regular test user
userPassword, _ := auth.HashPassword("password123")
user := models.User{
Username: "testuser",
Email:    "testuser@example.com",
Password: userPassword,
IsAdmin:  false,
}
if err := db.Create(&user).Error; err != nil {
log.Fatal("Failed to create test user:", err)
}
fmt.Println("Created test user: testuser")
}

fmt.Println("Test users setup complete!")
}

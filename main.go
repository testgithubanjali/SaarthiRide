package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
	"SaarthiRide/internal/routes"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to PostgreSQL
	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}

	log.Println("PostgreSQL connected successfully!")

	// Create or update database tables
	if err := db.AutoMigrate(
		&models.User{},
		&models.Vehicle{},
		&models.Ride{},
	); err != nil {
		log.Fatal("Database migration failed: ", err)
	}

	log.Println("Database tables migrated successfully!")

	// Set up API routes
	router := routes.SetupRoutes()

	// Get application port
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	log.Printf("SaarthiGo server starting on port %s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Server failed: ", err)
	}
}

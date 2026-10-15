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
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to PostgreSQL
	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}

	log.Println("PostgreSQL connected successfully!")

	// Create database tables from models
	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("Database migration failed: ", err)
	}

	log.Println("User table created successfully!")

	// Set up routes
	router := routes.SetupRoutes()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("SaarthiGo server starting on port %s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Server failed: ", err)
	}
}

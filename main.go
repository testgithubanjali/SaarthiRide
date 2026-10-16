package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
	"SaarthiRide/internal/routes"
)

// @title SaarthiRide API
// @version 1.0
// @description REST API for the SaarthiRide ride-booking application.
// @host localhost:8080
// @BasePath /api
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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

	// Prevent a vehicle from being assigned to multiple active rides.
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_rides_unique_active_vehicle
		ON public.rides (vehicle_id)
		WHERE vehicle_id IS NOT NULL
		  AND status IN ('accepted', 'in_progress');
	`).Error; err != nil {
		log.Fatal("Failed to create active vehicle assignment index: ", err)
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

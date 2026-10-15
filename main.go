package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"auto-rickshaw-backend/internal/routes"
)

func main() {
	log.Println("Loading environment configuration")
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	} else {
		log.Println("Environment configuration loaded")
	}

	log.Println("Initializing routes")
	router := routes.SetupRoutes()
	log.Println("Routes initialized")

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting SaarthiGo server on :%s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

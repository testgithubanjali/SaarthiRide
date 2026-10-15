package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"

	"auto-rickshaw-backend/internal/routes"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	router := routes.SetupRoutes()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("SaarthiGo server starting on port %s", port)

	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

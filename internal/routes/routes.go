package routes

import (
	"SaarthiRide/internal/handlers"
	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {

	router := gin.Default()

	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
	}

	return router
}

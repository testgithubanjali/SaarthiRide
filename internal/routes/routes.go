package routes

import (
	"SaarthiRide/internal/handlers"
	"SaarthiRide/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {

	router := gin.Default()

	api := router.Group("/api")
	{
		api.GET("/health", handlers.HealthCheck)
		api.POST("/register", handlers.Register)
		api.POST("/login", handlers.Login)
	}
	api.GET(
		"/profile",
		middleware.AuthRequired(),
		handlers.GetProfile,
	)
	return router
}

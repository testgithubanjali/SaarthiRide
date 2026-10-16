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

		api.GET("/profile", middleware.AuthRequired(), handlers.GetProfile)
		api.PUT("/profile", middleware.AuthRequired(), handlers.UpdateProfile)

		api.POST("/vehicles", middleware.AuthRequired(), handlers.CreateVehicle)
		api.GET("/vehicles", middleware.AuthRequired(), handlers.GetMyVehicles)

		api.PATCH(
			"/vehicles/:id/availability",
			middleware.AuthRequired(),
			handlers.UpdateVehicleAvailability,
		)
		api.PATCH(
			"/vehicles/:id/location",
			middleware.AuthRequired(),
			handlers.UpdateDriverLocation,
		)
		api.GET("/vehicles/nearby", handlers.GetNearbyVehicles)

		// Ride booking
		api.POST("/rides", middleware.AuthRequired(), handlers.CreateRide)
		api.GET("/rides/pending", middleware.AuthRequired(), handlers.GetPendingRides)
		api.PATCH("/rides/:id/accept", middleware.AuthRequired(), handlers.AcceptRide)

		// Start an accepted ride
		api.PATCH("/rides/:id/start", middleware.AuthRequired(), handlers.StartRide)
		api.PATCH(
			"/rides/:id/complete",
			middleware.AuthRequired(),
			handlers.CompleteRide,
		)
		api.GET(
			"/rides/history",
			middleware.AuthRequired(),
			handlers.GetRideHistory,
		)
		api.PATCH(
			"/rides/:id/cancel",
			middleware.AuthRequired(),
			handlers.CancelRide,
		)
	}

	return router
}

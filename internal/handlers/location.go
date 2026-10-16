package handlers

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
)

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

// UpdateDriverLocation godoc
// @Summary      Update driver location
// @Description  Updates the location of an available vehicle belonging to the authenticated driver.
// @Tags         Vehicles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path int true "Vehicle ID"
// @Param        request body UpdateLocationRequest true "Current vehicle coordinates"
// @Success      200 {object} map[string]interface{} "Vehicle location updated successfully"
// @Failure      400 {object} map[string]interface{} "Invalid latitude or longitude"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      404 {object} map[string]interface{} "Vehicle not found for this driver"
// @Failure      409 {object} map[string]interface{} "Vehicle must be available before sharing location"
// @Failure      500 {object} map[string]interface{} "Database error"
// @Router       /vehicles/{id}/location [patch]
func UpdateDriverLocation(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required.",
		})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid authenticated user.",
		})
		return
	}

	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Latitude == nil || req.Longitude == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Valid latitude and longitude are required.",
		})
		return
	}

	latitude := *req.Latitude
	longitude := *req.Longitude
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Latitude must be between -90 and 90.",
		})
		return
	}
	if math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Longitude must be between -180 and 180.",
		})
		return
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Database connection failed.",
		})
		return
	}

	var vehicle models.Vehicle
	if err := db.Where(
		"id = ? AND driver_id = ?",
		c.Param("id"),
		userID,
	).First(&vehicle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "Vehicle not found for this driver.",
		})
		return
	}

	if !vehicle.IsAvailable {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Make the vehicle available before sharing its location.",
		})
		return
	}

	if err := db.Model(&vehicle).Updates(map[string]interface{}{
		"latitude":  latitude,
		"longitude": longitude,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not update vehicle location.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Vehicle location updated successfully!",
		"vehicle": gin.H{
			"id":        vehicle.ID,
			"latitude":  latitude,
			"longitude": longitude,
		},
	})
}

package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
)

type CreateVehicleRequest struct {
	RegistrationNumber string `json:"registration_number" binding:"required"`
	Model              string `json:"model" binding:"required"`
	Capacity           int    `json:"capacity" binding:"required,min=1,max=10"`
}

// CreateVehicle godoc
// @Summary      Register a vehicle
// @Description  Registers a vehicle for the authenticated driver.
// @Tags         Vehicles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body CreateVehicleRequest true "Vehicle details"
// @Success      201 {object} map[string]interface{} "Vehicle registered successfully"
// @Failure      400 {object} map[string]interface{} "Invalid vehicle details"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Only drivers can register vehicles"
// @Failure      409 {object} map[string]interface{} "Registration number may already exist"
// @Failure      500 {object} map[string]interface{} "Database error"
// @Router       /vehicles [post]
func CreateVehicle(c *gin.Context) {
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

	var req CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Provide a registration number, model, and capacity between 1 and 10.",
		})
		return
	}

	req.RegistrationNumber = strings.ToUpper(
		strings.TrimSpace(req.RegistrationNumber),
	)
	req.Model = strings.TrimSpace(req.Model)

	if req.RegistrationNumber == "" || req.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Registration number and model cannot be empty.",
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

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "User not found.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not verify driver.",
		})
		return
	}

	if user.Role != "driver" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Only drivers can register vehicles.",
		})
		return
	}

	vehicle := models.Vehicle{
		DriverID:           user.ID,
		RegistrationNumber: req.RegistrationNumber,
		Model:              req.Model,
		Capacity:           req.Capacity,
	}

	if err := db.Create(&vehicle).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Could not register vehicle. The registration number may already exist.",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Vehicle registered successfully!",
		"vehicle": vehicle,
	})
}

// GetMyVehicles godoc
// @Summary      Get my vehicles
// @Description  Returns vehicles registered by the authenticated user.
// @Tags         Vehicles
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{} "Vehicles retrieved successfully"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      500 {object} map[string]interface{} "Could not retrieve vehicles"
// @Router       /vehicles [get]
func GetMyVehicles(c *gin.Context) {
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

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Database connection failed.",
		})
		return
	}

	var vehicles []models.Vehicle
	if err := db.Where("driver_id = ?", userID).
		Order("created_at DESC").
		Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not retrieve vehicles.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"vehicles": vehicles,
	})
}

type UpdateAvailabilityRequest struct {
	IsAvailable *bool `json:"is_available" binding:"required"`
}

// UpdateVehicleAvailability godoc
// @Summary      Update vehicle availability
// @Description  Updates the availability status of a vehicle registered by the authenticated driver.
// @Tags         Vehicles
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Vehicle ID"
// @Param        request body UpdateAvailabilityRequest true "Availability update details"
// @Success      200 {object} map[string]interface{} "Vehicle availability updated successfully"
// @Failure      400 {object} map[string]interface{} "Invalid availability status"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      404 {object} map[string]interface{} "Vehicle not found for this driver"
// @Failure      500 {object} map[string]interface{} "Could not update vehicle availability"
// @Router       /vehicles/{id}/availability [patch]
func UpdateVehicleAvailability(c *gin.Context) {
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

	var req UpdateAvailabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.IsAvailable == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "is_available must be true or false.",
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

	if err := db.Model(&vehicle).Update(
		"is_available",
		*req.IsAvailable,
	).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not update vehicle availability.",
		})
		return
	}

	vehicle.IsAvailable = *req.IsAvailable

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Vehicle availability updated successfully!",
		"vehicle": vehicle,
	})
}

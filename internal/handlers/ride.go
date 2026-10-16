package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CreateRideRequest struct {
	PickupLatitude       float64 `json:"pickup_latitude" binding:"required"`
	PickupLongitude      float64 `json:"pickup_longitude" binding:"required"`
	DestinationLatitude  float64 `json:"destination_latitude" binding:"required"`
	DestinationLongitude float64 `json:"destination_longitude" binding:"required"`
}

func validLatitude(value float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0) &&
		value >= -90 && value <= 90
}

func validLongitude(value float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0) &&
		value >= -180 && value <= 180
}

// CreateRide allows authenticated passengers to request a ride.
func CreateRide(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required.",
		})
		return
	}

	passengerID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid authenticated user.",
		})
		return
	}

	var request CreateRideRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Please provide valid pickup and destination coordinates.",
		})
		return
	}

	if !validLatitude(request.PickupLatitude) ||
		!validLatitude(request.DestinationLatitude) ||
		!validLongitude(request.PickupLongitude) ||
		!validLongitude(request.DestinationLongitude) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Coordinates are outside valid latitude or longitude ranges.",
		})
		return
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not connect to database.",
		})
		return
	}

	var user models.User
	if err := db.First(&user, passengerID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authenticated user was not found.",
		})
		return
	}

	if user.Role != "passenger" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Only passengers can request rides.",
		})
		return
	}

	ride := models.Ride{
		PassengerID:          user.ID,
		PickupLatitude:       request.PickupLatitude,
		PickupLongitude:      request.PickupLongitude,
		DestinationLatitude:  request.DestinationLatitude,
		DestinationLongitude: request.DestinationLongitude,
		Status:               "pending",
	}

	if err := db.Create(&ride).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not create ride request.",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Ride requested successfully!",
		"ride":    ride,
	})
}

// GetPendingRides allows authenticated drivers to view pending ride requests.
func GetPendingRides(c *gin.Context) {
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
			"message": "Could not connect to database.",
		})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authenticated user was not found.",
		})
		return
	}

	if user.Role != "driver" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Only drivers can view pending rides.",
		})
		return
	}

	var rides []models.Ride
	if err := db.Where("status = ?", "pending").
		Order("created_at ASC").
		Find(&rides).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not retrieve pending rides.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(rides),
		"rides":  rides,
	})
}

// AcceptRide assigns an available vehicle and driver to a pending ride.
// Both changes happen in one transaction.
func AcceptRide(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required.",
		})
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid authenticated user.",
		})
		return
	}

	rideID := c.Param("id")

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not connect to database.",
		})
		return
	}

	var driver models.User
	if err := db.First(&driver, driverID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Driver account not found.",
		})
		return
	}

	if driver.Role != "driver" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Only drivers can accept rides.",
		})
		return
	}

	var ride models.Ride

	err = db.Transaction(func(tx *gorm.DB) error {
		// Find an available vehicle belonging to this driver.
		var vehicle models.Vehicle
		if err := tx.Where(
			"driver_id = ? AND is_available = ?",
			driverID, true,
		).First(&vehicle).Error; err != nil {
			return err
		}

		// Claim the ride only if it is still pending and unassigned.
		result := tx.Model(&models.Ride{}).
			Where(
				"id = ? AND status = ? AND driver_id IS NULL",
				rideID, "pending",
			).
			Updates(map[string]interface{}{
				"driver_id":  driverID,
				"vehicle_id": vehicle.ID,
				"status":     "accepted",
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// Mark the vehicle unavailable.
		vehicleResult := tx.Model(&models.Vehicle{}).
			Where(
				"id = ? AND driver_id = ? AND is_available = ?",
				vehicle.ID, driverID, true,
			).
			Update("is_available", false)

		if vehicleResult.Error != nil {
			return vehicleResult.Error
		}
		if vehicleResult.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return tx.First(&ride, "id = ?", rideID).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Ride or available vehicle is no longer available.",
			})
			return
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "No available vehicle was found for this driver.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not accept ride.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ride accepted successfully!",
		"ride":    ride,
	})
}

// StartRide allows the assigned driver to start an accepted ride.
func StartRide(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required.",
		})
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid authenticated user.",
		})
		return
	}

	rideID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || rideID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid ride ID.",
		})
		return
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not connect to database.",
		})
		return
	}

	var driver models.User
	if err := db.First(&driver, driverID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Driver account not found.",
		})
		return
	}

	if driver.Role != "driver" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Only drivers can start rides.",
		})
		return
	}

	result := db.Model(&models.Ride{}).
		Where("id = ? AND driver_id = ? AND status = ?", rideID, driverID, "accepted").
		Update("status", "in_progress")
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not start ride.",
		})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Ride is not accepted by this driver or is no longer available to start.",
		})
		return
	}

	var ride models.Ride
	if err := db.First(&ride, rideID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Ride started but could not be retrieved.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ride started successfully!",
		"ride":    ride,
	})
}

func CompleteRide(c *gin.Context) {
	userIDValue, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required.",
		})
		return
	}

	driverID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Invalid authenticated user.",
		})
		return
	}

	rideID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || rideID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid ride ID.",
		})
		return
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not connect to database.",
		})
		return
	}

	var driver models.User
	if err := db.First(&driver, driverID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Driver account not found.",
		})
		return
	}

	if driver.Role != "driver" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Only drivers can complete rides.",
		})
		return
	}

	var ride models.Ride

	err = db.Transaction(func(tx *gorm.DB) error {
		// Lock and load the ride.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ride, rideID).Error; err != nil {
			return err
		}

		// Only the assigned driver can complete this ride.
		if ride.DriverID == nil || *ride.DriverID != driverID {
			return errors.New("ride_not_assigned")
		}

		if ride.Status != "in_progress" {
			return errors.New("ride_not_in_progress")
		}

		// Complete the ride.
		if err := tx.Model(&ride).
			Update("status", "completed").Error; err != nil {
			return err
		}

		// Make the assigned vehicle available again.
		if ride.VehicleID != nil {
			result := tx.Model(&models.Vehicle{}).
				Where("id = ? AND driver_id = ? AND is_available = ?",
					*ride.VehicleID, driverID, false).
				Update("is_available", true)

			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return errors.New("assigned_vehicle_not_found_or_already_available")
			}
		}

		return tx.First(&ride, ride.ID).Error
	})

	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "Ride not found.",
			})
		case err.Error() == "ride_not_assigned":
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "You are not assigned to this ride.",
			})
		case err.Error() == "ride_not_in_progress":
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Only rides in progress can be completed.",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Could not complete ride. The transaction was rolled back.",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ride completed successfully!",
		"ride":    ride,
	})
}

func GetRideHistory(c *gin.Context) {
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
			"message": "Could not connect to database.",
		})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authenticated user was not found.",
		})
		return
	}

	var rides []models.Ride
	query := db.Order("created_at DESC")

	switch user.Role {
	case "passenger":
		query = query.Where("passenger_id = ?", user.ID)
	case "driver":
		query = query.Where("driver_id = ?", user.ID)
	default:
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Your account role cannot access ride history.",
		})
		return
	}

	if err := query.Find(&rides).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not retrieve ride history.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"count":  len(rides),
		"rides":  rides,
	})
}

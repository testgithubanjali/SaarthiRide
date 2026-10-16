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
//
// CreateRide godoc
// @Summary      Request a ride
// @Description  Creates a ride request for the authenticated passenger.
// @Tags         Rides
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body CreateRideRequest true "Pickup and destination coordinates"
// @Success      201 {object} map[string]interface{} "Ride requested successfully"
// @Failure      400 {object} map[string]interface{} "Invalid coordinates or request"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Only passengers can request rides"
// @Failure      500 {object} map[string]interface{} "Database error"
// @Router       /rides [post]
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
//
// GetPendingRides godoc
// @Summary      Get pending ride requests
// @Description  Returns pending rides for authenticated drivers.
// @Tags         Rides
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{} "Pending rides retrieved"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Only drivers can view pending rides"
// @Failure      500 {object} map[string]interface{} "Could not retrieve rides"
// @Router       /rides/pending [get]
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
//
// AcceptRide godoc
// @Summary      Accept a ride
// @Description  Accepts a pending ride and assigns an available vehicle to the driver.
// @Tags         Rides
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Ride ID"
// @Success      200 {object} map[string]interface{} "Ride accepted successfully"
// @Failure      400 {object} map[string]interface{} "Invalid ride ID"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Only drivers can accept rides"
// @Failure      404 {object} map[string]interface{} "Ride not found"
// @Failure      409 {object} map[string]interface{} "Ride or vehicle is unavailable"
// @Failure      500 {object} map[string]interface{} "Could not accept ride"
// @Router       /rides/{id}/accept [patch]
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Driver account not found.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not retrieve driver account.",
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

	// Distinguish a vehicle conflict from a ride conflict.
	errNoAvailableVehicle := errors.New("no_available_vehicle")
	errRideUnavailable := errors.New("ride_unavailable")

	var ride models.Ride

	err = db.Transaction(func(tx *gorm.DB) error {
		// Find and lock an available vehicle belonging to this driver.
		// Exclude vehicles assigned to accepted or ongoing rides.
		var vehicle models.Vehicle

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(`
				driver_id = ?
				AND is_available = ?
				AND NOT EXISTS (
					SELECT 1
					FROM rides
					WHERE rides.vehicle_id = vehicles.id
					AND rides.status IN (?, ?)
				)
			`, driverID, true, "accepted", "in_progress").
			Order("id ASC").
			First(&vehicle).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errNoAvailableVehicle
			}
			return err
		}

		// Accept the ride only if it is still pending and unassigned.
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
			return errRideUnavailable
		}

		// Mark the vehicle unavailable only if it is still available.
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
			return errNoAvailableVehicle
		}

		return tx.First(&ride, rideID).Error
	})

	if err != nil {
		switch {
		case errors.Is(err, errNoAvailableVehicle):
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "No available vehicle was found for this driver.",
			})
		case errors.Is(err, errRideUnavailable):
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Ride is no longer pending or has already been assigned.",
			})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "Ride not found.",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Could not accept ride.",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ride accepted successfully!",
		"ride":    ride,
	})
}

// StartRide allows the assigned driver to start an accepted ride.
//
// StartRide godoc
// @Summary      Start a ride
// @Description  Starts an accepted ride assigned to the authenticated driver.
// @Tags         Rides
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Ride ID"
// @Success      200 {object} map[string]interface{} "Ride started successfully"
// @Failure      400 {object} map[string]interface{} "Invalid ride ID"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Only drivers can start rides"
// @Failure      409 {object} map[string]interface{} "Ride cannot be started"
// @Failure      500 {object} map[string]interface{} "Could not start ride"
// @Router       /rides/{id}/start [patch]
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

// CompleteRide godoc
// @Summary      Complete a ride
// @Description  Completes an in-progress ride assigned to the authenticated driver and releases its vehicle.
// @Tags         Rides
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Ride ID"
// @Success      200 {object} map[string]interface{} "Ride completed successfully"
// @Failure      400 {object} map[string]interface{} "Invalid ride ID"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Driver is not assigned to the ride"
// @Failure      404 {object} map[string]interface{} "Ride not found"
// @Failure      409 {object} map[string]interface{} "Ride cannot be completed"
// @Failure      500 {object} map[string]interface{} "Could not complete ride"
// @Router       /rides/{id}/complete [patch]
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

// GetRideHistory godoc
// @Summary      Get ride history
// @Description  Returns rides belonging to the authenticated passenger or driver.
// @Tags         Rides
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} map[string]interface{} "Ride history retrieved successfully"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "Account role cannot access ride history"
// @Failure      500 {object} map[string]interface{} "Could not retrieve ride history"
// @Router       /rides/history [get]
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

// CancelRide godoc
// @Summary      Cancel a ride
// @Description  Cancels an eligible ride owned by the passenger or assigned to the driver.
// @Tags         Rides
// @Security     BearerAuth
// @Produce      json
// @Param        id path int true "Ride ID"
// @Success      200 {object} map[string]interface{} "Ride cancelled successfully"
// @Failure      400 {object} map[string]interface{} "Invalid ride ID"
// @Failure      401 {object} map[string]interface{} "Authentication required"
// @Failure      403 {object} map[string]interface{} "User is not authorized to cancel this ride"
// @Failure      404 {object} map[string]interface{} "Ride not found"
// @Failure      409 {object} map[string]interface{} "Ride can no longer be cancelled"
// @Failure      500 {object} map[string]interface{} "Could not cancel ride"
// @Router       /rides/{id}/cancel [patch]
func CancelRide(c *gin.Context) {
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

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authenticated user was not found.",
		})
		return
	}

	if user.Role != "passenger" && user.Role != "driver" {
		c.JSON(http.StatusForbidden, gin.H{
			"status":  "error",
			"message": "Your account cannot cancel rides.",
		})
		return
	}

	var ride models.Ride

	err = db.Transaction(func(tx *gorm.DB) error {
		// Lock the ride to prevent conflicting status changes.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&ride, rideID).Error; err != nil {
			return err
		}

		switch user.Role {
		case "passenger":
			if ride.PassengerID != user.ID {
				return errors.New("not_ride_owner")
			}

			if ride.Status != "pending" && ride.Status != "accepted" {
				return errors.New("ride_not_cancellable")
			}

		case "driver":
			if ride.DriverID == nil || *ride.DriverID != user.ID {
				return errors.New("not_assigned_driver")
			}

			if ride.Status != "accepted" {
				return errors.New("ride_not_cancellable")
			}
		}

		// Release the assigned vehicle when an accepted ride
		// is cancelled.
		if ride.Status == "accepted" && ride.VehicleID != nil {
			if ride.DriverID == nil {
				return errors.New("invalid_assignment")
			}

			result := tx.Model(&models.Vehicle{}).
				Where(
					"id = ? AND driver_id = ?",
					*ride.VehicleID,
					*ride.DriverID,
				).
				Update("is_available", true)

			if result.Error != nil {
				return result.Error
			}

			if result.RowsAffected == 0 {
				return errors.New("assigned_vehicle_not_found")
			}
		}

		// Cancel only if the status still matches.
		result := tx.Model(&models.Ride{}).
			Where("id = ? AND status = ?", ride.ID, ride.Status).
			Update("status", "cancelled")

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("ride_status_changed")
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
		case err.Error() == "not_ride_owner":
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "You can only cancel your own rides.",
			})
		case err.Error() == "not_assigned_driver":
			c.JSON(http.StatusForbidden, gin.H{
				"status":  "error",
				"message": "You are not assigned to this ride.",
			})
		case err.Error() == "ride_not_cancellable":
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "This ride can no longer be cancelled.",
			})
		case err.Error() == "ride_status_changed":
			c.JSON(http.StatusConflict, gin.H{
				"status":  "error",
				"message": "Ride status changed. Please try again.",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Could not cancel ride. Changes were rolled back.",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Ride cancelled successfully!",
		"ride":    ride,
	})
}

package handlers

import (
	"math"
	"net/http"
	"sort"
	"strconv"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
	"SaarthiRide/internal/utils"

	"github.com/gin-gonic/gin"
)

type NearbyVehicleResult struct {
	ID                 uint    `json:"id"`
	DriverID           uint    `json:"driver_id"`
	RegistrationNumber string  `json:"registration_number"`
	Model              string  `json:"model"`
	Capacity           int     `json:"capacity"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	DistanceKM         float64 `json:"distance_km"`
}

func GetNearbyVehicles(c *gin.Context) {
	latitude, err := strconv.ParseFloat(c.Query("latitude"), 64)
	if err != nil || math.IsNaN(latitude) || math.IsInf(latitude, 0) ||
		latitude < -90 || latitude > 90 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Valid latitude between -90 and 90 is required.",
		})
		return
	}

	longitude, err := strconv.ParseFloat(c.Query("longitude"), 64)
	if err != nil || math.IsNaN(longitude) || math.IsInf(longitude, 0) ||
		longitude < -180 || longitude > 180 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Valid longitude between -180 and 180 is required.",
		})
		return
	}

	radius := 5.0
	if c.Query("radius") != "" {
		radius, err = strconv.ParseFloat(c.Query("radius"), 64)
		if err != nil || math.IsNaN(radius) || math.IsInf(radius, 0) ||
			radius <= 0 || radius > 100 {
			c.JSON(http.StatusBadRequest, gin.H{
				"status":  "error",
				"message": "Radius must be greater than 0 and at most 100 km.",
			})
			return
		}
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not connect to database.",
		})
		return
	}

	var vehicles []models.Vehicle
	if err := db.Where(
		"is_available = ? AND latitude IS NOT NULL AND longitude IS NOT NULL",
		true,
	).Find(&vehicles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not retrieve vehicles.",
		})
		return
	}

	nearby := make([]NearbyVehicleResult, 0)

	for _, vehicle := range vehicles {
		if vehicle.Latitude == nil || vehicle.Longitude == nil {
			continue
		}

		distance := utils.CalculateDistance(
			latitude,
			longitude,
			*vehicle.Latitude,
			*vehicle.Longitude,
		)

		if distance <= radius {
			nearby = append(nearby, NearbyVehicleResult{
				ID:                 vehicle.ID,
				DriverID:           vehicle.DriverID,
				RegistrationNumber: vehicle.RegistrationNumber,
				Model:              vehicle.Model,
				Capacity:           vehicle.Capacity,
				Latitude:           *vehicle.Latitude,
				Longitude:          *vehicle.Longitude,
				DistanceKM:         math.Round(distance*100) / 100,
			})
		}
	}

	sort.Slice(nearby, func(i, j int) bool {
		return nearby[i].DistanceKM < nearby[j].DistanceKM
	})

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"count":     len(nearby),
		"radius_km": radius,
		"vehicles":  nearby,
	})
}

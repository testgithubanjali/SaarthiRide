package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
)

type UpdateProfileRequest struct {
	Name string `json:"name" binding:"required"`
}

func UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Name is required.",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Name cannot be empty.",
		})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Authentication required.",
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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"status":  "error",
				"message": "User not found.",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not retrieve user.",
		})
		return
	}

	user.Name = req.Name

	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not update profile.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Profile updated successfully!",
		"user": gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"phone":      user.Phone,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

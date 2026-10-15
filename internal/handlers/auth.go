package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"SaarthiRide/internal/config"
	"SaarthiRide/internal/models"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid input. Password must contain at least 8 characters.",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))

	if req.Name == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Name and phone are required.",
		})
		return
	}

	if req.Role != "passenger" && req.Role != "driver" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Role must be passenger or driver.",
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not secure password.",
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

	user := models.User{
		Name:         req.Name,
		Phone:        req.Phone,
		PasswordHash: string(passwordHash),
		Role:         req.Role,
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"status":  "error",
			"message": "Registration failed. This phone number may already be registered.",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Registration successful!",
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Name,
			"phone": user.Phone,
			"role":  user.Role,
		},
	})
}

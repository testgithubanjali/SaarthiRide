package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck godoc
// @Summary      Check API health
// @Description  Checks whether the SaarthiGo backend API is running.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  map[string]interface{}  "API is running successfully"
// @Router       /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "SaarthiGo backend is running!",
	})
}

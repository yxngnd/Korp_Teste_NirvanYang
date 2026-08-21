package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health trata GET /health.
// Usado pelo docker-compose healthcheck e para verificação manual rápida
// de que o serviço subiu corretamente.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

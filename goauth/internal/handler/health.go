package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查 Handler
type HealthHandler struct{}

// NewHealthHandler 创建健康检查 Handler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health 健康检查
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": "1.0.0",
	})
}

// Ready 就绪检查
func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

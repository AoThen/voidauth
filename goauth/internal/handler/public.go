package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goauth/internal/config"
	"goauth/internal/util"
)

// PublicHandler 公开 Handler
type PublicHandler struct {
	cfg *config.Config
}

// NewPublicHandler 创建公开 Handler
func NewPublicHandler(cfg *config.Config) *PublicHandler {
	return &PublicHandler{cfg: cfg}
}

// GetConfig 获取公开配置
func (h *PublicHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"appName":       h.cfg.UI.AppName,
		"appColor":      h.cfg.UI.AppColor,
		"signupEnabled": h.cfg.UI.SignupEnabled,
	})
}

// CheckPasswordStrength 检查密码强度
func (h *PublicHandler) CheckPasswordStrength(c *gin.Context) {
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	score := util.PasswordScore(req.Password)
	err := util.CheckPasswordStrength(req.Password, h.cfg.Security.PasswordMin, h.cfg.Security.PasswordMinScore)

	c.JSON(http.StatusOK, gin.H{
		"score":   score,
		"valid":   err == nil,
		"message": errorMessage(err),
	})
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	switch err {
	case util.ErrPasswordTooShort:
		return "密码太短"
	case util.ErrPasswordTooWeak:
		return "密码强度不足"
	default:
		return err.Error()
	}
}

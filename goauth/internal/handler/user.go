package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"goauth/internal/config"
	"goauth/internal/model"
	"goauth/internal/repo"
	"goauth/internal/service"
)

// UserHandler 用户 Handler
type UserHandler struct {
	userService *service.UserService
	totpService *service.TotpService
	sessionRepo *repo.SessionRepo
	cfg         *config.Config
}

// NewUserHandler 创建用户 Handler
func NewUserHandler(userService *service.UserService, totpService *service.TotpService, sessionRepo *repo.SessionRepo, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userService: userService,
		totpService: totpService,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

// GetMe 获取当前用户
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("userID")

	user, err := h.userService.GetByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	// 检查 TOTP 状态
	hasTotp, _ := h.totpService.IsEnabled(c.Request.Context(), userID.(string))
	user.HasTotp = hasTotp

	c.JSON(http.StatusOK, user)
}

// UpdateProfile 更新资料
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req struct {
		Name  *string `json:"name"`
		Email *string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	user, err := h.userService.UpdateProfile(c.Request.Context(), userID.(string), req.Name, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdatePassword 更新密码
func (h *UserHandler) UpdatePassword(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	err := h.userService.UpdatePassword(c.Request.Context(), userID.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码已更新"})
}

// SetupTotp 设置 TOTP
func (h *UserHandler) SetupTotp(c *gin.Context) {
	userID, _ := c.Get("userID")
	username, _ := c.Get("username")

	resp, err := h.totpService.Setup(c.Request.Context(), userID.(string), username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// VerifyTotp 验证并确认 TOTP 设置
func (h *UserHandler) VerifyTotp(c *gin.Context) {
	userID, _ := c.Get("userID")

	var req struct {
		Code            string `json:"code" binding:"required"`
		Secret          string `json:"secret"`          // 明文密钥（用于验证）
		EncryptedSecret string `json:"encryptedSecret"` // 加密密钥（用于存储）
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求"})
		return
	}

	// 验证验证码（使用明文密钥）
	valid := h.totpService.VerifySetupCode(req.Secret, req.Code)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码无效"})
		return
	}

	// 验证成功，存储密钥
	err := h.totpService.ConfirmSetup(c.Request.Context(), userID.(string), req.EncryptedSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 检查是否需要升级 session（从 pwd-mfa-setup-required 升级为 pwd）
	mfaSetupRequired, _ := c.Get("mfaSetupRequired")
	if mfaSetupRequired == true {
		// 从 cookie 或 header 获取 token
		token, err := c.Cookie("session")
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}
		if token != "" {
			// 查找并升级 session
			session, err := h.sessionRepo.FindByToken(c.Request.Context(), token)
			if err == nil && session != nil && session.AMR == "pwd-mfa-setup-required" {
				// 升级 session
				var expiresAt time.Time
				if session.RememberMe {
					expiresAt = time.Now().Add(h.cfg.Session.TTLRemember)
				} else {
					expiresAt = time.Now().Add(h.cfg.Session.TTL)
				}
				session.AMR = "pwd"
				session.ExpiresAt = model.CustomTime{Time: expiresAt}
				_ = h.sessionRepo.Update(c.Request.Context(), session)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// RemoveTotp 移除 TOTP
func (h *UserHandler) RemoveTotp(c *gin.Context) {
	userID, _ := c.Get("userID")

	err := h.totpService.Remove(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "TOTP 已移除"})
}

// GetSessions 获取所有 Session
func (h *UserHandler) GetSessions(c *gin.Context) {
	userID, _ := c.Get("userID")

	sessions, err := h.userService.GetUserSessions(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话失败"})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// TerminateSession 终止指定 Session
func (h *UserHandler) TerminateSession(c *gin.Context) {
	sessionID := c.Param("id")

	err := h.userService.TerminateSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终止会话失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "会话已终止"})
}

// ListUsers 列出用户（管理员）
func (h *UserHandler) ListUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, count, err := h.userService.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": count,
	})
}

// GetUser 获取用户（管理员）
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

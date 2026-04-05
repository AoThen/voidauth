package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"goauth/internal/config"
	"goauth/internal/service"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	authService *service.AuthService
	cfg         *config.Config
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(authService *service.AuthService, cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{authService: authService, cfg: cfg}
}

// RequireAuth 需要登录
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 cookie 或 header 获取 token
		token, err := c.Cookie("session")
		if err != nil {
			// 尝试从 Authorization header 获取
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 验证 session
		user, session, err := m.authService.ValidateSessionWithAMR(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "会话无效或已过期"})
			c.Abort()
			return
		}

		// 检查 AMR 状态：拒绝待 TOTP 验证的 session 访问受保护资源
		if session != nil && session.AMR == "pwd-totp-pending" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要完成二步验证"})
			c.Abort()
			return
		}

		// 检查 AMR 状态：拒绝需要设置 MFA 的 session 访问受保护资源
		if session != nil && session.AMR == "pwd-mfa-setup-required" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要设置二步验证", "requireMfaSetup": true})
			c.Abort()
			return
		}

		// 滑动过期：刷新 session 过期时间
		if session != nil {
			newExpiresAt, refreshErr := m.authService.RefreshSession(c.Request.Context(), session)
			if refreshErr == nil && newExpiresAt != session.ExpiresAt.Time {
				// Session 已续期，更新 cookie
				m.updateSessionCookie(c, token, newExpiresAt)
			}
		}

		// 设置用户信息到 context
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("isAdmin", user.IsAdmin)
		c.Set("user", user)
		if session != nil {
			c.Set("sessionID", session.ID)
		}

		c.Next()
	}
}

// updateSessionCookie 更新 session cookie 的过期时间
func (m *AuthMiddleware) updateSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	secure := m.cfg.IsCookieSecure()
	sameSite := m.cfg.GetCookieSameSite()
	domain := m.cfg.Server.CookieDomain

	var sameSiteValue http.SameSite
	switch sameSite {
	case "strict":
		sameSiteValue = http.SameSiteStrictMode
	case "none":
		sameSiteValue = http.SameSiteNoneMode
	default:
		sameSiteValue = http.SameSiteLaxMode
	}

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge > 0 {
		c.SetSameSite(sameSiteValue)
		c.SetCookie("session", token, maxAge, "/", domain, secure, true)
	}
}

// RequireMfaSetup 需要设置 MFA（允许 pwd-mfa-setup-required 状态的用户访问）
// 用于 TOTP 设置相关 API
func (m *AuthMiddleware) RequireMfaSetup() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 cookie 或 header 获取 token
		token, err := c.Cookie("session")
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			c.Abort()
			return
		}

		// 验证 session
		user, session, err := m.authService.ValidateSessionWithAMR(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "会话无效或已过期"})
			c.Abort()
			return
		}

		// 只允许 pwd-mfa-setup-required 或正常登录状态
		if session != nil && session.AMR == "pwd-totp-pending" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要完成二步验证"})
			c.Abort()
			return
		}

		// 设置用户信息到 context
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("isAdmin", user.IsAdmin)
		c.Set("user", user)
		// 标记是否需要升级 session（从 pwd-mfa-setup-required 升级为 pwd）
		if session != nil && session.AMR == "pwd-mfa-setup-required" {
			c.Set("mfaSetupRequired", true)
		}

		c.Next()
	}
}

// RequireAdmin 需要管理员权限
func (m *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("isAdmin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// OptionalAuth 可选认证
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token != "" {
			user, err := m.authService.ValidateSession(c.Request.Context(), token)
			if err == nil {
				c.Set("userID", user.ID)
				c.Set("username", user.Username)
				c.Set("isAdmin", user.IsAdmin)
				c.Set("user", user)
			}
		}

		c.Next()
	}
}

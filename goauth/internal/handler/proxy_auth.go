package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"goauth/internal/repo"
	"goauth/internal/service"
)

// ProxyAuthHandler ProxyAuth Handler
type ProxyAuthHandler struct {
	authService   *service.AuthService
	proxyAuthRepo *repo.ProxyAuthRepo
	groupRepo     *repo.GroupRepo
}

// NewProxyAuthHandler 创建 ProxyAuth Handler
func NewProxyAuthHandler(
	authService *service.AuthService,
	proxyAuthRepo *repo.ProxyAuthRepo,
	groupRepo *repo.GroupRepo,
) *ProxyAuthHandler {
	return &ProxyAuthHandler{
		authService:   authService,
		proxyAuthRepo: proxyAuthRepo,
		groupRepo:     groupRepo,
	}
}

// ForwardAuth Traefik ForwardAuth 端点
// 验证用户是否已登录，以及是否有权限访问指定域名
func (h *ProxyAuthHandler) ForwardAuth(c *gin.Context) {
	// 从请求中获取目标域名
	domain := c.Query("domain")
	if domain == "" {
		// 从 Host header 提取域名
		host := c.Request.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		domain = host
	}

	// 获取 session token
	token, err := c.Cookie("session")
	if err != nil || token == "" {
		// 尝试从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 验证 session
	user, err := h.authService.ValidateSession(c.Request.Context(), token)
	if err != nil || user == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 检查用户是否被禁用
	if user.Disabled {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 查找 ProxyAuth 配置
	proxyAuth, err := h.proxyAuthRepo.FindByDomain(c.Request.Context(), domain)
	if err != nil {
		// 没有配置则允许所有已登录用户
		c.Header("X-User-Id", user.ID)
		c.Header("X-User-Name", user.Username)
		c.Status(http.StatusOK)
		return
	}

	// 检查 MFA 要求
	if proxyAuth.MFARequired && !user.MFARequired {
		// 该域名需要 MFA，但用户未启用
		// 可以选择允许或拒绝，这里选择允许（因为用户可能已经通过 MFA 登录）
	}

	// 检查分组限制
	groupIDs, err := h.proxyAuthRepo.GetProxyAuthGroups(c.Request.Context(), proxyAuth.ID)
	if err == nil && len(groupIDs) > 0 {
		// 有分组限制，检查用户是否在允许的分组中
		userGroups, err := h.groupRepo.GetUserGroups(c.Request.Context(), user.ID)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		allowed := false
		for _, ug := range userGroups {
			for _, ag := range groupIDs {
				if ug.ID == ag {
					allowed = true
					break
				}
			}
			if allowed {
				break
			}
		}

		if !allowed {
			c.Status(http.StatusForbidden)
			return
		}
	}

	// 返回用户信息给反向代理
	c.Header("X-User-Id", user.ID)
	c.Header("X-User-Name", user.Username)
	if user.Email != nil {
		c.Header("X-User-Email", *user.Email)
	}
	c.Status(http.StatusOK)
}

// AuthRequest Nginx Auth Request 端点
// 与 ForwardAuth 类似，但遵循 Nginx auth_request 模块的约定
func (h *ProxyAuthHandler) AuthRequest(c *gin.Context) {
	// Nginx auth_request 通常通过 original_uri 传递原始请求信息
	// 这里我们主要检查 session

	// 从 cookie 获取 session token
	token, err := c.Cookie("session")
	if err != nil || token == "" {
		// 尝试从 Authorization header 获取
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 验证 session
	user, err := h.authService.ValidateSession(c.Request.Context(), token)
	if err != nil || user == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 检查用户是否被禁用
	if user.Disabled {
		c.Status(http.StatusUnauthorized)
		return
	}

	// 设置响应头供 Nginx 使用
	c.Header("X-User-Id", user.ID)
	c.Header("X-User-Name", user.Username)
	c.Status(http.StatusOK)
}

package handler

import (
	"github.com/gin-gonic/gin"

	"goauth/internal/oidc"
)

// OIDCHandler OIDC Handler
type OIDCHandler struct {
	provider *oidc.Provider
}

// NewOIDCHandler 创建 OIDC Handler
func NewOIDCHandler(provider *oidc.Provider) *OIDCHandler {
	return &OIDCHandler{provider: provider}
}

// Handler 返回 OIDC 处理器
func (h *OIDCHandler) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		h.provider.Handler().ServeHTTP(c.Writer, c.Request)
	}
}

// Authorize 授权端点
func (h *OIDCHandler) Authorize(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

// Token Token 端点
func (h *OIDCHandler) Token(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

// UserInfo 用户信息端点
func (h *OIDCHandler) UserInfo(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

// Introspect 令牌内省端点
func (h *OIDCHandler) Introspect(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

// Revoke 令牌撤销端点
func (h *OIDCHandler) Revoke(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

// EndSession 登出端点
func (h *OIDCHandler) EndSession(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

// JWKS JWKS 端点
func (h *OIDCHandler) JWKS(c *gin.Context) {
	h.provider.Handler().ServeHTTP(c.Writer, c.Request)
}

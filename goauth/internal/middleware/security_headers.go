package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders 安全响应头中间件
// 添加标准安全响应头以防护常见攻击
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// 启用浏览器 XSS 过滤器
		c.Header("X-XSS-Protection", "1; mode=block")

		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 权限策略（替代 Feature-Policy）
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Content-Security-Policy 根据环境配置
		// 注意：CSP 配置较复杂，需要根据实际前端资源调整
		// 这里使用相对宽松的策略，生产环境应更严格
		// c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		c.Next()
	}
}

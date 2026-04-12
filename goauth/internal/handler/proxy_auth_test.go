package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ========== sanitizeHeader 安全函数测试 ==========

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "正常字符串不变",
			input:    "normaluser",
			expected: "normaluser",
		},
		{
			name:     "包含 \\r 被移除",
			input:    "user\rname",
			expected: "username",
		},
		{
			name:     "包含 \\n 被移除",
			input:    "user\nname",
			expected: "username",
		},
		{
			name:     "包含 \\r\\n 被移除",
			input:    "user\r\nname",
			expected: "username",
		},
		{
			name:     "多个换行符被移除",
			input:    "user\n\r\nname",
			expected: "username",
		},
		{
			name:     "HTTP Header 注入攻击 - 添加恶意 header",
			input:    "user\r\nX-Admin: true",
			expected: "userX-Admin: true",
		},
		{
			name:     "HTTP Header 注入攻击 - Cookie 窃取",
			input:    "user\r\nSet-Cookie: session=evil",
			expected: "userSet-Cookie: session=evil",
		},
		{
			name:     "空字符串",
			input:    "",
			expected: "",
		},
		{
			name:     "只有换行符",
			input:    "\r\n\r\n",
			expected: "",
		},
		{
			name:     "包含特殊字符但不含换行",
			input:    "user@example.com",
			expected: "user@example.com",
		},
		{
			name:     "中文用户名",
			input:    "张三",
			expected: "张三",
		},
		{
			name:     "带空格的用户名",
			input:    "john doe",
			expected: "john doe",
		},
		{
			name:     "CRLF 注入 - 多行攻击",
			input:    "admin\r\nSet-Cookie: evil=true\r\nX-Injected: bad",
			expected: "adminSet-Cookie: evil=trueX-Injected: bad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHeader(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeHeader_Placeholder 测试 sanitizeHeader 在 ProxyAuthHandler 中的使用
// 这是一个占位测试，确保导出函数的行为符合预期
func TestSanitizeHeader_Placeholder(t *testing.T) {
	// 确保函数存在且可调用
	assert.Equal(t, "test", sanitizeHeader("test"))
	assert.Equal(t, "", sanitizeHeader("\r\n"))
}
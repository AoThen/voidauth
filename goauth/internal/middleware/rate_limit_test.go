package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(3, time.Minute)

	// 前三次应该允许
	for i := 0; i < 3; i++ {
		assert.True(t, limiter.Allow("192.168.1.1"), "Request %d should be allowed", i+1)
	}

	// 第四次应该被拒绝
	assert.False(t, limiter.Allow("192.168.1.1"), "Request 4 should be blocked")
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	// 不同 IP 应该独立计数
	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.False(t, limiter.Allow("192.168.1.1")) // 第一个 IP 达到限制

	assert.True(t, limiter.Allow("192.168.1.2")) // 第二个 IP 应该独立计数
	assert.True(t, limiter.Allow("192.168.1.2"))
	assert.False(t, limiter.Allow("192.168.1.2"))
}

func TestRateLimiter_WindowReset(t *testing.T) {
	// 使用很短的窗口进行测试
	limiter := NewRateLimiter(2, 100*time.Millisecond)

	// 消耗配额
	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.True(t, limiter.Allow("192.168.1.1"))
	assert.False(t, limiter.Allow("192.168.1.1"))

	// 等待窗口过期
	time.Sleep(150 * time.Millisecond)

	// 应该重置
	assert.True(t, limiter.Allow("192.168.1.1"), "Should be allowed after window reset")
}

func TestRateLimit_Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(2, time.Minute)

	router := gin.New()
	router.Use(RateLimit(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 前两次请求应该成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 第三次请求应该被限制
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, 429, w.Code, "Request 3 should be rate limited")
}

func TestRateLimit_Middleware_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(1, time.Minute)

	router := gin.New()
	router.Use(RateLimit(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// 第一个 IP 的请求
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "192.168.1.1:1234"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 第一个 IP 的第二个请求应该被限制
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	assert.Equal(t, 429, w2.Code)

	// 第二个 IP 的请求应该成功
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "192.168.1.2:1234"
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
}

func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewRateLimiter(100, time.Minute)

	// 并发测试
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				limiter.Allow("192.168.1.1")
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证总共 100 次请求后应该被限制
	assert.False(t, limiter.Allow("192.168.1.1"), "Should be blocked after 100 requests")
}

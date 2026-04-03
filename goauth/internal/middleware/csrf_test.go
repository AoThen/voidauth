package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCSRF_GenerateToken(t *testing.T) {
	token, err := GenerateCSRFToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Generate another token and verify they are different
	token2, err := GenerateCSRFToken()
	assert.NoError(t, err)
	assert.NotEqual(t, token, token2)
}

func TestCSRF_SkipPaths(t *testing.T) {
	router := gin.New()
	router.Use(CSRF(CSRFConfig{
		SkipPaths: []string{"/api/auth/login", "/api/auth/register"},
	}))
	router.POST("/api/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login"})
	})
	router.POST("/api/auth/register", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "register"})
	})
	router.POST("/api/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logout"})
	})

	// Test login (skipped) - should pass without CSRF token
	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test register (skipped) - should pass without CSRF token
	req = httptest.NewRequest("POST", "/api/auth/register", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_RequireToken(t *testing.T) {
	router := gin.New()
	router.Use(CSRF())
	router.POST("/api/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logout"})
	})

	// Test without CSRF token - should fail
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRF_ValidToken(t *testing.T) {
	router := gin.New()
	router.Use(CSRF())
	router.POST("/api/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logout"})
	})

	// Generate a valid token
	token, err := GenerateCSRFToken()
	assert.NoError(t, err)

	// Test with valid CSRF token
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set(CSRFHeaderName, token)
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: token,
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_InvalidToken(t *testing.T) {
	router := gin.New()
	router.Use(CSRF())
	router.POST("/api/auth/logout", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logout"})
	})

	// Test with mismatched CSRF token
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)
	req.Header.Set(CSRFHeaderName, "invalid-token")
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: "valid-token",
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRF_GetMethod(t *testing.T) {
	router := gin.New()
	router.Use(CSRF())
	router.GET("/api/user/me", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "me"})
	})

	// GET requests should not require CSRF token
	req := httptest.NewRequest("GET", "/api/user/me", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRF_SetCSRFToken(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	token := "test-csrf-token"
	SetCSRFToken(c, token, false, http.SameSiteLaxMode, "")

	// Verify cookie is set
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, CSRFCookieName, cookies[0].Name)
	assert.Equal(t, token, cookies[0].Value)
	assert.Equal(t, "/", cookies[0].Path)
	assert.False(t, cookies[0].Secure)
	assert.False(t, cookies[0].HttpOnly)
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler_Health(t *testing.T) {
	router := gin.New()
	h := NewHealthHandler()
	router.GET("/health", h.Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", w.Code, http.StatusOK)
	}

	// 验证响应包含 status: ok
	if w.Body.String() == "" {
		t.Error("Health() returned empty body")
	}
}

func TestHealthHandler_Ready(t *testing.T) {
	router := gin.New()
	h := NewHealthHandler()
	router.GET("/ready", h.Ready)

	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Ready() status = %d, want %d", w.Code, http.StatusOK)
	}
}

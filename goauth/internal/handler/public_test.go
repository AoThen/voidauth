package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"goauth/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func makeTestConfig() *config.Config {
	return &config.Config{
		UI: config.UIConfig{
			AppName:       "TestApp",
			AppColor:      "#906bc7",
			SignupEnabled: true,
		},
		Security: config.SecurityConfig{
			PasswordMin:      8,
			PasswordMinScore: 3,
		},
	}
}

func TestPublicHandler_GetConfig(t *testing.T) {
	router := gin.New()
	cfg := makeTestConfig()
	h := NewPublicHandler(cfg)
	router.GET("/api/public/config", h.GetConfig)

	req, _ := http.NewRequest("GET", "/api/public/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetConfig() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["appName"] != "TestApp" {
		t.Errorf("GetConfig() appName = %v, want TestApp", resp["appName"])
	}
}

func TestPublicHandler_CheckPasswordStrength(t *testing.T) {
	router := gin.New()
	cfg := makeTestConfig()
	h := NewPublicHandler(cfg)
	router.POST("/api/public/password-strength", h.CheckPasswordStrength)

	tests := []struct {
		name     string
		password string
		wantCode int
	}{
		{"有效密码", "MyStr0ng!Pass123456", http.StatusOK},
		{"太短", "short", http.StatusOK}, // 返回 valid: false
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"password": tt.password})
			req, _ := http.NewRequest("POST", "/api/public/password-strength", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("CheckPasswordStrength() status = %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}

func TestPublicHandler_CheckPasswordStrength_EmptyBody(t *testing.T) {
	router := gin.New()
	cfg := makeTestConfig()
	h := NewPublicHandler(cfg)
	router.POST("/api/public/password-strength", h.CheckPasswordStrength)

	req, _ := http.NewRequest("POST", "/api/public/password-strength", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("CheckPasswordStrength() with empty body status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

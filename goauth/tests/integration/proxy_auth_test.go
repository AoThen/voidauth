package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goauth/internal/model"
	"goauth/tests/helper"
)

// ========== ProxyAuth 管理测试 ==========

// TestProxyAuth_Create_Success 测试成功创建 ProxyAuth
func TestProxyAuth_Create_Success(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建 ProxyAuth
	reqBody := map[string]interface{}{
		"domain":          "app.example.com",
		"mfaRequired":     false,
		"maxSessionLength": 24,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", server.Server.URL+"/api/admin/proxy-auth", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "创建 ProxyAuth 应该成功，响应: %s", string(respBody))

	var result model.ProxyAuth
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "app.example.com", result.Domain)
	assert.False(t, result.MFARequired)
}

// TestProxyAuth_Create_WithGroups 测试带分组的 ProxyAuth
func TestProxyAuth_Create_WithGroups(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建分组
	group := server.CreateGroup(t, helper.WithGroupName("restricted-group"))

	// 创建 ProxyAuth 并关联分组
	reqBody := map[string]interface{}{
		"domain":      "restricted.example.com",
		"mfaRequired": true,
		"groupIds":    []string{group.ID},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", server.Server.URL+"/api/admin/proxy-auth", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// TestProxyAuth_Create_Unauthorized 测试非管理员无法创建
func TestProxyAuth_Create_Unauthorized(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建普通用户
	_, token := server.LoginAsUser(t, helper.WithUsername("normaluser"))

	// 尝试创建 ProxyAuth
	reqBody := map[string]interface{}{
		"domain": "unauthorized.example.com",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", server.Server.URL+"/api/admin/proxy-auth", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// TestProxyAuth_List 测试列出 ProxyAuth
func TestProxyAuth_List(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建多个 ProxyAuth
	proxyAuth1 := &model.ProxyAuth{
		Domain:      "list1.example.com",
		CreatedBy:   admin.ID,
	}
	proxyAuth2 := &model.ProxyAuth{
		Domain:      "list2.example.com",
		CreatedBy:   admin.ID,
	}
	err := server.ProxyAuthRepo.Create(context.Background(), proxyAuth1, nil)
	require.NoError(t, err)
	err = server.ProxyAuthRepo.Create(context.Background(), proxyAuth2, nil)
	require.NoError(t, err)

	// 获取列表
	req, _ := http.NewRequest("GET", server.Server.URL+"/api/admin/proxy-auth", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result []*model.ProxyAuth
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(result), 2)
}

// TestProxyAuth_Update 测试更新 ProxyAuth
func TestProxyAuth_Update(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建 ProxyAuth
	proxyAuth := &model.ProxyAuth{
		Domain:      "update.example.com",
		CreatedBy:   admin.ID,
	}
	err := server.ProxyAuthRepo.Create(context.Background(), proxyAuth, nil)
	require.NoError(t, err)

	// 更新 ProxyAuth
	reqBody := map[string]interface{}{
		"domain":          "updated.example.com",
		"mfaRequired":     true,
		"maxSessionLength": 48,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PATCH", server.Server.URL+"/api/admin/proxy-auth/"+proxyAuth.ID, bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证更新
	updated, err := server.ProxyAuthRepo.FindByID(context.Background(), proxyAuth.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated.example.com", updated.Domain)
	assert.True(t, updated.MFARequired)
}

// TestProxyAuth_Delete 测试删除 ProxyAuth
func TestProxyAuth_Delete(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建 ProxyAuth
	proxyAuth := &model.ProxyAuth{
		Domain:      "delete.example.com",
		CreatedBy:   admin.ID,
	}
	err := server.ProxyAuthRepo.Create(context.Background(), proxyAuth, nil)
	require.NoError(t, err)

	// 删除
	req, _ := http.NewRequest("DELETE", server.Server.URL+"/api/admin/proxy-auth/"+proxyAuth.ID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证已删除
	_, err = server.ProxyAuthRepo.FindByID(context.Background(), proxyAuth.ID)
	assert.Error(t, err, "ProxyAuth 应该已删除")
}

// ========== ForwardAuth 测试 ==========

// TestForwardAuth_Success 测试成功认证
func TestForwardAuth_Success(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建并登录用户
	_, token := server.LoginAsUser(t, helper.WithUsername("forwarduser"))

	// 测试 ForwardAuth
	req, _ := http.NewRequest("GET", server.Server.URL+"/authz/forward-auth", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-User-Id"))
	assert.NotEmpty(t, resp.Header.Get("X-User-Name"))
}

// TestForwardAuth_Unauthorized 测试未登录用户
func TestForwardAuth_Unauthorized(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 不带 cookie 测试
	resp, err := http.Get(server.Server.URL + "/authz/forward-auth")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestForwardAuth_WithBearerToken 测试使用 Bearer Token
func TestForwardAuth_WithBearerToken(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建并登录用户
	_, token := server.LoginAsUser(t, helper.WithUsername("beareruser"))

	// 使用 Bearer Token
	req, _ := http.NewRequest("GET", server.Server.URL+"/authz/forward-auth", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestForwardAuth_DisabledUser 测试禁用用户
func TestForwardAuth_DisabledUser(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建并登录用户
	user, token := server.LoginAsUser(t, helper.WithUsername("disabledforward"))

	// 更新为禁用状态
	user.Disabled = true
	err := server.UserRepo.Update(context.Background(), user)
	require.NoError(t, err)

	// 测试 ForwardAuth
	req, _ := http.NewRequest("GET", server.Server.URL+"/authz/forward-auth", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestForwardAuth_WithDomainRestriction 测试域名限制
func TestForwardAuth_WithDomainRestriction(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员
	admin := server.CreateAdmin(t)

	// 创建分组
	group := server.CreateGroup(t, helper.WithGroupName("allowed-group"))

	// 创建用户并加入分组
	user := server.CreateUser(t, helper.WithUsername("groupuser"))
	err := server.GroupRepo.AddUserToGroup(context.Background(), user.ID, group.ID)
	require.NoError(t, err)

	// 登录
	token := server.Login(t, "groupuser", "My-Test-Pass-2024-Secure!")

	// 创建 ProxyAuth 限制域名并关联分组
	proxyAuth := &model.ProxyAuth{
		Domain:      "restricted.example.com",
		MFARequired: false,
		CreatedBy:   admin.ID,
	}
	err = server.ProxyAuthRepo.Create(context.Background(), proxyAuth, []string{group.ID})
	require.NoError(t, err)

	// 测试带域名参数的 ForwardAuth（用户在分组中）
	req, _ := http.NewRequest("GET", server.Server.URL+"/authz/forward-auth?domain=restricted.example.com", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestForwardAuth_GroupForbidden 测试分组禁止访问
func TestForwardAuth_GroupForbidden(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员
	admin := server.CreateAdmin(t)

	// 创建分组
	group := server.CreateGroup(t, helper.WithGroupName("exclusive-group"))

	// 创建用户（不在分组中）
	_, token := server.LoginAsUser(t, helper.WithUsername("outsideruser"))

	// 创建 ProxyAuth 限制域名并关联分组
	proxyAuth := &model.ProxyAuth{
		Domain:      "exclusive.example.com",
		MFARequired: false,
		CreatedBy:   admin.ID,
	}
	err := server.ProxyAuthRepo.Create(context.Background(), proxyAuth, []string{group.ID})
	require.NoError(t, err)

	// 测试带域名参数的 ForwardAuth（用户不在分组中）
	req, _ := http.NewRequest("GET", server.Server.URL+"/authz/forward-auth?domain=exclusive.example.com", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ========== AuthRequest 测试 ==========

// TestAuthRequest_Success 测试 Nginx AuthRequest 成功
func TestAuthRequest_Success(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建并登录用户
	_, token := server.LoginAsUser(t, helper.WithUsername("authrequser"))

	// 测试 AuthRequest
	req, _ := http.NewRequest("GET", server.Server.URL+"/authz/auth-request", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("X-User-Id"))
	assert.NotEmpty(t, resp.Header.Get("X-User-Name"))
}

// TestAuthRequest_Unauthorized 测试 Nginx AuthRequest 未认证
func TestAuthRequest_Unauthorized(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 不带 cookie 测试
	resp, err := http.Get(server.Server.URL + "/authz/auth-request")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ========== ProxyAuth Repo 测试 ==========

// TestProxyAuth_Repo_FindByDomain 测试按域名查找
func TestProxyAuth_Repo_FindByDomain(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	admin := server.CreateAdmin(t)

	// 创建 ProxyAuth
	proxyAuth := &model.ProxyAuth{
		Domain:      "find.example.com",
		MFARequired: true,
		CreatedBy:   admin.ID,
	}
	err := server.ProxyAuthRepo.Create(context.Background(), proxyAuth, nil)
	require.NoError(t, err)

	// 按域名查找
	found, err := server.ProxyAuthRepo.FindByDomain(context.Background(), "find.example.com")
	require.NoError(t, err)
	assert.Equal(t, proxyAuth.ID, found.ID)
	assert.True(t, found.MFARequired)

	// 查找不存在的域名
	_, err = server.ProxyAuthRepo.FindByDomain(context.Background(), "notfound.example.com")
	assert.Error(t, err)
}

// TestProxyAuth_Repo_GetProxyAuthGroups 测试获取关联分组
func TestProxyAuth_Repo_GetProxyAuthGroups(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	admin := server.CreateAdmin(t)

	// 创建分组
	group1 := server.CreateGroup(t, helper.WithGroupName("pgroup1"))
	group2 := server.CreateGroup(t, helper.WithGroupName("pgroup2"))

	// 创建 ProxyAuth 并关联分组
	proxyAuth := &model.ProxyAuth{
		Domain:      "groups.example.com",
		CreatedBy:   admin.ID,
	}
	err := server.ProxyAuthRepo.Create(context.Background(), proxyAuth, []string{group1.ID, group2.ID})
	require.NoError(t, err)

	// 获取关联分组
	groupIDs, err := server.ProxyAuthRepo.GetProxyAuthGroups(context.Background(), proxyAuth.ID)
	require.NoError(t, err)
	assert.Len(t, groupIDs, 2)
}

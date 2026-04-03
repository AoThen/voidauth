package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goauth/internal/model"
	"goauth/tests/helper"
)

// ========== 邀请创建测试 ==========

// TestInvitation_Create_Success 测试成功创建邀请
func TestInvitation_Create_Success(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建邀请
	reqBody := map[string]interface{}{
		"email":         "invited@example.com",
		"username":      "inviteduser",
		"name":          "Invited User",
		"emailVerified": true,
		"expiresIn":     72,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", server.Server.URL+"/api/admin/invitations", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "创建邀请应该成功，响应: %s", string(respBody))

	var result model.Invitation
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	assert.NotEmpty(t, result.ID, "应该返回邀请 ID")
	assert.Equal(t, "invited@example.com", *result.Email)
	assert.Equal(t, "inviteduser", *result.Username)
	assert.True(t, result.EmailVerified)
	assert.True(t, result.ExpiresAt.After(time.Now()))
}

// TestInvitation_Create_WithGroups 测试带分组的邀请
func TestInvitation_Create_WithGroups(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建分组
	group := server.CreateGroup(t, helper.WithGroupName("test-group"))

	// 创建邀请并关联分组
	reqBody := map[string]interface{}{
		"email":     "groupuser@example.com",
		"groupIds":  []string{group.ID},
		"expiresIn": 48,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", server.Server.URL+"/api/admin/invitations", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

// TestInvitation_Create_Unauthorized 测试非管理员无法创建邀请
func TestInvitation_Create_Unauthorized(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建普通用户并登录
	_, token := server.LoginAsUser(t, helper.WithUsername("normaluser"))

	// 尝试创建邀请
	reqBody := map[string]interface{}{
		"email": "test@example.com",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", server.Server.URL+"/api/admin/invitations", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: token})
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ========== 邀请列表测试 ==========

// TestInvitation_List 测试列出邀请
func TestInvitation_List(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建多个邀请
	email1 := "invite1@example.com"
	email2 := "invite2@example.com"
	_, err := server.InvitationService.CreateInvitation(
		context.Background(),
		&email1,
		nil,
		nil,
		false,
		nil,
		admin.ID,
		72,
	)
	require.NoError(t, err)
	_, err = server.InvitationService.CreateInvitation(
		context.Background(),
		&email2,
		nil,
		nil,
		false,
		nil,
		admin.ID,
		72,
	)
	require.NoError(t, err)

	// 获取邀请列表
	req, _ := http.NewRequest("GET", server.Server.URL+"/api/admin/invitations", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result []*model.Invitation
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, len(result), 2)
}

// ========== 邀请删除测试 ==========

// TestInvitation_Delete 测试删除邀请
func TestInvitation_Delete(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	// 创建管理员并登录
	admin := server.CreateAdmin(t)
	token := server.Login(t, admin.Username, "My-Test-Pass-2024-Secure!")

	// 创建邀请
	email := "delete@example.com"
	invitation, err := server.InvitationService.CreateInvitation(
		context.Background(),
		&email,
		nil,
		nil,
		false,
		nil,
		admin.ID,
		72,
	)
	require.NoError(t, err)

	// 删除邀请
	req, _ := http.NewRequest("DELETE", server.Server.URL+"/api/admin/invitations/"+invitation.ID, nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: token})

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证已删除
	invitations, _ := server.InvitationService.ListInvitations(context.Background())
	for _, inv := range invitations {
		assert.NotEqual(t, invitation.ID, inv.ID, "邀请应该已删除")
	}
}

// ========== 邀请服务层测试 ==========

// TestInvitation_Service_Validate 测试邀请验证
func TestInvitation_Service_Validate(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	admin := server.CreateAdmin(t)

	// 创建有效邀请
	email := "valid@example.com"
	invitation, err := server.InvitationService.CreateInvitation(
		context.Background(),
		&email,
		nil,
		nil,
		false,
		nil,
		admin.ID,
		72,
	)
	require.NoError(t, err)

	// 验证有效邀请
	_, groupIDs, err := server.InvitationService.ValidateInvitation(context.Background(), invitation.Challenge)
	require.NoError(t, err)
	assert.Empty(t, groupIDs)

	// 验证无效邀请
	_, _, err = server.InvitationService.ValidateInvitation(context.Background(), "invalid-challenge")
	assert.Error(t, err)
}

// TestInvitation_Service_CleanupExpired 测试清理过期邀请
func TestInvitation_Service_CleanupExpired(t *testing.T) {
	server := helper.NewTestServer(t)
	defer server.Close()

	admin := server.CreateAdmin(t)

	// 创建过期邀请
	email := "cleanup@example.com"
	expiredInvitation := &model.Invitation{
		Email:         &email,
		Challenge:     "cleanup-challenge",
		EmailVerified: false,
		CreatedBy:     admin.ID,
		ExpiresAt:     model.CustomTime{Time: time.Now().Add(-24 * time.Hour)},
		CreatedAt:     model.Now(),
	}
	err := server.InvitationRepo.Create(context.Background(), expiredInvitation, nil)
	require.NoError(t, err)

	// 清理过期邀请
	count, err := server.InvitationService.CleanupExpired(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, int64(1))
}
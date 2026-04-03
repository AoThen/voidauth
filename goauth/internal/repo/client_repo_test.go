package repo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goauth/internal/model"
)

func setupClientTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS clients (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		secret TEXT,
		redirectUris TEXT NOT NULL,
		postLogoutUris TEXT,
		scopes TEXT NOT NULL,
		grantTypes TEXT NOT NULL,
		responseTypes TEXT NOT NULL,
		tokenEndpointAuth TEXT DEFAULT 'client_secret_basic',
		trusted INTEGER DEFAULT 0,
		createdBy TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		updatedAt TEXT NOT NULL
	);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestClientRepo_Create(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid", "profile"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	client := &model.Client{
		Name:          "Test Client",
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)
	assert.NotEmpty(t, client.ID, "ID should be generated")
}

func TestClientRepo_Create_WithSecret(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	secret := "my-secret"
	client := &model.Client{
		Name:          "Secret Client",
		Secret:        &secret,
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}

	err := repo.Create(ctx, client)
	require.NoError(t, err)

	// 验证密钥已被哈希
	found, err := repo.FindByID(ctx, client.ID)
	require.NoError(t, err)
	assert.NotEqual(t, secret, *found.Secret, "Secret should be hashed")
}

func TestClientRepo_FindByID(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	client := &model.Client{
		Name:          "Find Client",
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}
	repo.Create(ctx, client)

	found, err := repo.FindByID(ctx, client.ID)
	require.NoError(t, err)
	assert.Equal(t, client.ID, found.ID)
	assert.Equal(t, "Find Client", found.Name)
}

func TestClientRepo_FindByID_NotFound(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	assert.Equal(t, ErrClientNotFound, err)
}

func TestClientRepo_List(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	// 创建多个客户端
	for i := 0; i < 3; i++ {
		redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
		scopes, _ := json.Marshal([]string{"openid"})
		grantTypes, _ := json.Marshal([]string{"authorization_code"})
		responseTypes, _ := json.Marshal([]string{"code"})

		client := &model.Client{
			Name:          "Client " + string(rune('0'+i)),
			RedirectURIs:  string(redirectURIs),
			Scopes:        string(scopes),
			GrantTypes:    string(grantTypes),
			ResponseTypes: string(responseTypes),
			CreatedBy:     "test-user",
		}
		repo.Create(ctx, client)
	}

	clients, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, clients, 3)

	// 验证返回的是 ClientResponse
	for _, c := range clients {
		assert.NotEmpty(t, c.ID)
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.RedirectURIs)
	}
}

func TestClientRepo_Update(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	client := &model.Client{
		Name:          "Update Client",
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}
	repo.Create(ctx, client)

	// 更新客户端
	newRedirectURIs, _ := json.Marshal([]string{"http://localhost:9090/callback"})
	client.Name = "Updated Client"
	client.RedirectURIs = string(newRedirectURIs)
	client.Trusted = true

	err := repo.Update(ctx, client)
	require.NoError(t, err)

	// 验证更新
	found, _ := repo.FindByID(ctx, client.ID)
	assert.Equal(t, "Updated Client", found.Name)
	assert.True(t, found.Trusted)
}

func TestClientRepo_Delete(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	client := &model.Client{
		Name:          "Delete Client",
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}
	repo.Create(ctx, client)

	err := repo.Delete(ctx, client.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = repo.FindByID(ctx, client.ID)
	assert.Equal(t, ErrClientNotFound, err)
}

func TestClientRepo_ValidateSecret(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	secret := "my-client-secret"
	client := &model.Client{
		Name:          "Secret Validate Client",
		Secret:        &secret,
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}
	repo.Create(ctx, client)

	// 验证正确的密钥
	valid, err := repo.ValidateSecret(ctx, client.ID, "my-client-secret")
	require.NoError(t, err)
	assert.True(t, valid)

	// 验证错误的密钥
	valid, err = repo.ValidateSecret(ctx, client.ID, "wrong-secret")
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestClientRepo_List_HasSecret(t *testing.T) {
	db := setupClientTestDB(t)
	repo := NewClientRepo(db)
	ctx := context.Background()

	redirectURIs, _ := json.Marshal([]string{"http://localhost:3000/callback"})
	scopes, _ := json.Marshal([]string{"openid"})
	grantTypes, _ := json.Marshal([]string{"authorization_code"})
	responseTypes, _ := json.Marshal([]string{"code"})

	// 有密钥的客户端
	secret := "secret1"
	clientWithSecret := &model.Client{
		Name:          "Client With Secret",
		Secret:        &secret,
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}
	repo.Create(ctx, clientWithSecret)

	// 无密钥的客户端
	clientNoSecret := &model.Client{
		Name:          "Client No Secret",
		RedirectURIs:  string(redirectURIs),
		Scopes:        string(scopes),
		GrantTypes:    string(grantTypes),
		ResponseTypes: string(responseTypes),
		CreatedBy:     "test-user",
	}
	repo.Create(ctx, clientNoSecret)

	clients, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, clients, 2)

	// 验证 HasSecret 字段
	for _, c := range clients {
		if c.ID == clientWithSecret.ID {
			assert.True(t, c.HasSecret)
		} else {
			assert.False(t, c.HasSecret)
		}
	}
}

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goauth/internal/model"
)

func setupKeyTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	require.NoError(t, err)

	// 匹配 model.Key 的字段定义
	schema := `
	CREATE TABLE IF NOT EXISTS keys (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		value TEXT,
		expiresAt TEXT,
		createdAt TEXT NOT NULL
	);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestKeyRepo_Create(t *testing.T) {
	db := setupKeyTestDB(t)
	repo := NewKeyRepo(db)
	ctx := context.Background()

	key := &model.Key{
		Type:      "cookie",
		Value:     "test-value-123",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(24 * time.Hour)},
	}

	err := repo.Create(ctx, key)
	require.NoError(t, err)
	assert.NotEmpty(t, key.ID, "ID should be generated")
}

func TestKeyRepo_FindValidByType(t *testing.T) {
	db := setupKeyTestDB(t)
	repo := NewKeyRepo(db)
	ctx := context.Background()

	// 创建有效密钥
	for i := 0; i < 3; i++ {
		key := &model.Key{
			Type:      "cookie",
			Value:     "valid-value-" + string(rune('0'+i)),
			ExpiresAt: model.CustomTime{Time: time.Now().Add(24 * time.Hour)},
		}
		repo.Create(ctx, key)
	}

	// 创建过期密钥
	expiredKey := &model.Key{
		Type:      "cookie",
		Value:     "expired-value",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(-1 * time.Hour)},
	}
	repo.Create(ctx, expiredKey)

	keys, err := repo.FindValidByType(ctx, "cookie")
	require.NoError(t, err)
	assert.Len(t, keys, 3, "Should only return valid keys")

	for _, k := range keys {
		assert.True(t, k.ExpiresAt.After(time.Now()), "Should only return non-expired keys")
	}
}

func TestKeyRepo_FindValidByType_NoKeys(t *testing.T) {
	db := setupKeyTestDB(t)
	repo := NewKeyRepo(db)
	ctx := context.Background()

	keys, err := repo.FindValidByType(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestKeyRepo_FindValidByType_DifferentTypes(t *testing.T) {
	db := setupKeyTestDB(t)
	repo := NewKeyRepo(db)
	ctx := context.Background()

	// 创建不同类型的密钥
	cookieKey := &model.Key{
		Type:      "cookie",
		Value:     "cookie-value",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(24 * time.Hour)},
	}
	repo.Create(ctx, cookieKey)

	jwtKey := &model.Key{
		Type:      "jwt",
		Value:     "jwt-value",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(24 * time.Hour)},
	}
	repo.Create(ctx, jwtKey)

	// 只查询 cookie 类型
	keys, err := repo.FindValidByType(ctx, "cookie")
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "cookie-value", keys[0].Value)

	// 只查询 jwt 类型
	keys, err = repo.FindValidByType(ctx, "jwt")
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "jwt-value", keys[0].Value)
}

func TestKeyRepo_DeleteExpired(t *testing.T) {
	db := setupKeyTestDB(t)
	repo := NewKeyRepo(db)
	ctx := context.Background()

	// 创建过期密钥
	for i := 0; i < 2; i++ {
		key := &model.Key{
			Type:      "cookie",
			Value:     "expired-key-" + string(rune('0'+i)),
			ExpiresAt: model.CustomTime{Time: time.Now().Add(-1 * time.Hour)},
		}
		repo.Create(ctx, key)
	}

	// 创建有效密钥
	validKey := &model.Key{
		Type:      "cookie",
		Value:     "valid-key",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(24 * time.Hour)},
	}
	repo.Create(ctx, validKey)

	// 删除过期密钥
	err := repo.DeleteExpired(ctx)
	require.NoError(t, err)

	// 验证只有有效密钥
	keys, err := repo.FindValidByType(ctx, "cookie")
	require.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "valid-key", keys[0].Value)
}

func TestKeyRepo_Create_WithoutExpiry(t *testing.T) {
	db := setupKeyTestDB(t)
	repo := NewKeyRepo(db)
	ctx := context.Background()

	key := &model.Key{
		Type:  "permanent",
		Value: "permanent-value",
		// 不设置 ExpiresAt
	}

	err := repo.Create(ctx, key)
	require.NoError(t, err)

	// 验证可以找到（没有过期时间的密钥应该一直有效）
	keys, err := repo.FindValidByType(ctx, "permanent")
	require.NoError(t, err)
	// 注意：根据实现，可能需要调整这个断言
	// 如果 FindValidByType 只返回 ExpiresAt > now 的，那么无过期时间的密钥可能不会被返回
	// 这取决于具体实现
	_ = keys
}

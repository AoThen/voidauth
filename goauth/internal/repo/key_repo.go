package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"goauth/internal/model"
)

var ErrKeyNotFound = errors.New("key not found")

type KeyRepo struct {
	db *sqlx.DB
}

func NewKeyRepo(db *sqlx.DB) *KeyRepo {
	return &KeyRepo{db: db}
}

// Create 创建密钥
func (r *KeyRepo) Create(ctx context.Context, key *model.Key) error {
	if key.ID == "" {
		key.ID = uuid.NewString()
	}
	key.CreatedAt = model.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO keys (id, type, value, expiresAt, createdAt)
		VALUES (?, ?, ?, ?, ?)
	`, key.ID, key.Type, key.Value, key.ExpiresAt, key.CreatedAt)

	return err
}

// FindValidByType 查找有效的密钥
func (r *KeyRepo) FindValidByType(ctx context.Context, keyType string) ([]*model.Key, error) {
	var keys []*model.Key
	err := r.db.SelectContext(ctx, &keys, `
		SELECT * FROM keys WHERE type = ? AND expiresAt > ? ORDER BY createdAt DESC
	`, keyType, time.Now())
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// DeleteExpired 删除过期密钥
func (r *KeyRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM keys WHERE expiresAt < ?`, time.Now())
	return err
}

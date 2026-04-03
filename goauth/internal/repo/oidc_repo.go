package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"goauth/internal/model"
)

var ErrPayloadNotFound = errors.New("payload not found")

type OIDCRepo struct {
	db *sqlx.DB
}

func NewOIDCRepo(db *sqlx.DB) *OIDCRepo {
	return &OIDCRepo{db: db}
}

// Create 创建 OIDC Payload
func (r *OIDCRepo) Create(ctx context.Context, payload *model.OIDCPayload) error {
	if payload.ID == "" {
		payload.ID = uuid.NewString()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO oidc_payloads (id, type, payload, grantId, userCode, uid, expiresAt, consumedAt, accountId)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, payload.ID, payload.Type, payload.Payload, payload.GrantID, payload.UserCode, payload.UID, payload.ExpiresAt, payload.ConsumedAt, payload.AccountID)

	return err
}

// FindByIDAndType 根据 ID 和类型查找
func (r *OIDCRepo) FindByIDAndType(ctx context.Context, id, payloadType string) (*model.OIDCPayload, error) {
	var payload model.OIDCPayload
	err := r.db.GetContext(ctx, &payload, `
		SELECT * FROM oidc_payloads WHERE id = ? AND type = ?
	`, id, payloadType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPayloadNotFound
		}
		return nil, err
	}
	return &payload, nil
}

// FindByAccountIDAndType 根据 AccountID 和类型查找
func (r *OIDCRepo) FindByAccountIDAndType(ctx context.Context, accountID, payloadType string) ([]*model.OIDCPayload, error) {
	var payloads []*model.OIDCPayload
	err := r.db.SelectContext(ctx, &payloads, `
		SELECT * FROM oidc_payloads WHERE accountId = ? AND type = ?
	`, accountID, payloadType)
	return payloads, err
}

// FindByAccountID 根据 AccountID 查找所有 payloads
func (r *OIDCRepo) FindByAccountID(ctx context.Context, accountID string) ([]*model.OIDCPayload, error) {
	var payloads []*model.OIDCPayload
	err := r.db.SelectContext(ctx, &payloads, `
		SELECT * FROM oidc_payloads WHERE accountId = ?
	`, accountID)
	return payloads, err
}

// Update 更新 OIDC Payload
func (r *OIDCRepo) Update(ctx context.Context, payload *model.OIDCPayload) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE oidc_payloads SET payload = ?, grantId = ?, userCode = ?, uid = ?, expiresAt = ?, consumedAt = ?, accountId = ?
		WHERE id = ? AND type = ?
	`, payload.Payload, payload.GrantID, payload.UserCode, payload.UID, payload.ExpiresAt, payload.ConsumedAt, payload.AccountID, payload.ID, payload.Type)
	return err
}

// Delete 删除 OIDC Payload
func (r *OIDCRepo) Delete(ctx context.Context, id, payloadType string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oidc_payloads WHERE id = ? AND type = ?`, id, payloadType)
	return err
}

// ListByType 列出指定类型的 Payload
func (r *OIDCRepo) ListByType(ctx context.Context, payloadType string, limit, offset int) ([]*model.OIDCPayload, error) {
	var payloads []*model.OIDCPayload
	err := r.db.SelectContext(ctx, &payloads, `
		SELECT * FROM oidc_payloads WHERE type = ? LIMIT ? OFFSET ?
	`, payloadType, limit, offset)
	return payloads, err
}

// DeleteExpired 删除过期的 Payload
func (r *OIDCRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oidc_payloads WHERE expiresAt IS NOT NULL AND expiresAt < ?`, time.Now())
	return err
}

// Consume 标记为已消费
func (r *OIDCRepo) Consume(ctx context.Context, id, payloadType string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE oidc_payloads SET consumedAt = ? WHERE id = ? AND type = ?
	`, now, id, payloadType)
	return err
}

// FindByGrantID 根据 GrantID 查找
func (r *OIDCRepo) FindByGrantID(ctx context.Context, grantID string) ([]*model.OIDCPayload, error) {
	var payloads []*model.OIDCPayload
	err := r.db.SelectContext(ctx, &payloads, `
		SELECT * FROM oidc_payloads WHERE grantId = ?
	`, grantID)
	return payloads, err
}

// DeleteByGrantID 根据 GrantID 删除
func (r *OIDCRepo) DeleteByGrantID(ctx context.Context, grantID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM oidc_payloads WHERE grantId = ?`, grantID)
	return err
}

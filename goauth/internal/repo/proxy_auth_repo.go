package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"goauth/internal/model"
)

var ErrProxyAuthNotFound = errors.New("proxy auth not found")

type ProxyAuthRepo struct {
	db *sqlx.DB
}

func NewProxyAuthRepo(db *sqlx.DB) *ProxyAuthRepo {
	return &ProxyAuthRepo{db: db}
}

// Create 创建 ProxyAuth
func (r *ProxyAuthRepo) Create(ctx context.Context, proxyAuth *model.ProxyAuth, groupIDs []string) error {
	if proxyAuth.ID == "" {
		proxyAuth.ID = uuid.NewString()
	}
	now := model.Now()
	proxyAuth.CreatedAt = now
	proxyAuth.UpdatedAt = now

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO proxy_auth (id, domain, mfaRequired, maxSessionLength, createdBy, createdAt, updatedAt)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, proxyAuth.ID, proxyAuth.Domain, proxyAuth.MFARequired, proxyAuth.MaxSessionLength,
		proxyAuth.CreatedBy, proxyAuth.CreatedAt, proxyAuth.UpdatedAt)
	if err != nil {
		return err
	}

	// 关联分组
	for _, groupID := range groupIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO proxy_auth_groups (proxyAuthId, groupId) VALUES (?, ?)
		`, proxyAuth.ID, groupID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// FindByID 根据 ID 查找 ProxyAuth
func (r *ProxyAuthRepo) FindByID(ctx context.Context, id string) (*model.ProxyAuth, error) {
	var proxyAuth model.ProxyAuth
	err := r.db.GetContext(ctx, &proxyAuth, `SELECT * FROM proxy_auth WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProxyAuthNotFound
		}
		return nil, err
	}
	return &proxyAuth, nil
}

// FindByDomain 根据域名查找 ProxyAuth
func (r *ProxyAuthRepo) FindByDomain(ctx context.Context, domain string) (*model.ProxyAuth, error) {
	var proxyAuth model.ProxyAuth
	err := r.db.GetContext(ctx, &proxyAuth, `SELECT * FROM proxy_auth WHERE domain = ?`, domain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrProxyAuthNotFound
		}
		return nil, err
	}
	return &proxyAuth, nil
}

// Update 更新 ProxyAuth
func (r *ProxyAuthRepo) Update(ctx context.Context, proxyAuth *model.ProxyAuth, groupIDs []string) error {
	proxyAuth.UpdatedAt = model.Now()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE proxy_auth SET domain = ?, mfaRequired = ?, maxSessionLength = ?, updatedAt = ? WHERE id = ?
	`, proxyAuth.Domain, proxyAuth.MFARequired, proxyAuth.MaxSessionLength, proxyAuth.UpdatedAt, proxyAuth.ID)
	if err != nil {
		return err
	}

	// 更新分组关联：先删除旧的，再添加新的
	_, err = tx.ExecContext(ctx, `DELETE FROM proxy_auth_groups WHERE proxyAuthId = ?`, proxyAuth.ID)
	if err != nil {
		return err
	}

	for _, groupID := range groupIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO proxy_auth_groups (proxyAuthId, groupId) VALUES (?, ?)
		`, proxyAuth.ID, groupID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete 删除 ProxyAuth
func (r *ProxyAuthRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM proxy_auth WHERE id = ?`, id)
	return err
}

// List 列出 ProxyAuth
func (r *ProxyAuthRepo) List(ctx context.Context) ([]*model.ProxyAuth, error) {
	var proxyAuths []*model.ProxyAuth
	err := r.db.SelectContext(ctx, &proxyAuths, `SELECT * FROM proxy_auth ORDER BY domain`)
	return proxyAuths, err
}

// GetProxyAuthGroups 获取 ProxyAuth 关联的分组ID
func (r *ProxyAuthRepo) GetProxyAuthGroups(ctx context.Context, proxyAuthID string) ([]string, error) {
	var groupIDs []string
	err := r.db.SelectContext(ctx, &groupIDs, `SELECT groupId FROM proxy_auth_groups WHERE proxyAuthId = ?`, proxyAuthID)
	return groupIDs, err
}

// ProxyAuthWithGroups ProxyAuth 带分组 ID 列表
type ProxyAuthWithGroups struct {
	ID               string  `db:"id" json:"id"`
	Domain           string  `db:"domain" json:"domain"`
	MFARequired      bool    `db:"mfaRequired" json:"mfaRequired"`
	MaxSessionLength *int    `db:"maxSessionLength" json:"maxSessionLength"`
	CreatedBy        string  `db:"createdBy" json:"createdBy"`
	GroupIDs         *string `db:"groupIds" json:"-"` // 逗号分隔的分组 ID，可能为 NULL
}

// ListWithGroups 列出 ProxyAuth 及其分组（单次查询，避免 N+1）
func (r *ProxyAuthRepo) ListWithGroups(ctx context.Context) ([]*ProxyAuthWithGroups, error) {
	var results []*ProxyAuthWithGroups
	err := r.db.SelectContext(ctx, &results, `
		SELECT pa.id, pa.domain, pa.mfaRequired, pa.maxSessionLength, pa.createdBy,
			GROUP_CONCAT(pag.groupId) as groupIds
		FROM proxy_auth pa
		LEFT JOIN proxy_auth_groups pag ON pa.id = pag.proxyAuthId
		GROUP BY pa.id
		ORDER BY pa.domain
	`)
	return results, err
}

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

var ErrInvitationNotFound = errors.New("invitation not found")

type InvitationRepo struct {
	db *sqlx.DB
}

func NewInvitationRepo(db *sqlx.DB) *InvitationRepo {
	return &InvitationRepo{db: db}
}

// Create 创建邀请
func (r *InvitationRepo) Create(ctx context.Context, invitation *model.Invitation, groupIDs []string) error {
	if invitation.ID == "" {
		invitation.ID = uuid.NewString()
	}
	now := model.Now()
	invitation.CreatedAt = now

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO invitations (id, email, username, name, challenge, emailVerified, createdBy, createdAt, expiresAt)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, invitation.ID, invitation.Email, invitation.Username, invitation.Name, invitation.Challenge,
		invitation.EmailVerified, invitation.CreatedBy, invitation.CreatedAt, invitation.ExpiresAt)
	if err != nil {
		return err
	}

	// 关联分组
	for _, groupID := range groupIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO invitation_groups (invitationId, groupId) VALUES (?, ?)
		`, invitation.ID, groupID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// FindByID 根据 ID 查找邀请
func (r *InvitationRepo) FindByID(ctx context.Context, id string) (*model.Invitation, error) {
	var invitation model.Invitation
	err := r.db.GetContext(ctx, &invitation, `SELECT * FROM invitations WHERE id = ?`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

// FindByChallenge 根据 challenge 查找邀请（用于验证邀请链接）
func (r *InvitationRepo) FindByChallenge(ctx context.Context, challenge string) (*model.Invitation, error) {
	var invitation model.Invitation
	err := r.db.GetContext(ctx, &invitation, `
		SELECT * FROM invitations WHERE challenge = ? AND expiresAt > ?
	`, challenge, time.Now())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return &invitation, nil
}

// Delete 删除邀请
func (r *InvitationRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM invitations WHERE id = ?`, id)
	return err
}

// List 列出邀请
func (r *InvitationRepo) List(ctx context.Context) ([]*model.Invitation, error) {
	var invitations []*model.Invitation
	err := r.db.SelectContext(ctx, &invitations, `SELECT * FROM invitations ORDER BY createdAt DESC`)
	return invitations, err
}

// GetInvitationGroups 获取邀请关联的分组ID
func (r *InvitationRepo) GetInvitationGroups(ctx context.Context, invitationID string) ([]string, error) {
	var groupIDs []string
	err := r.db.SelectContext(ctx, &groupIDs, `SELECT groupId FROM invitation_groups WHERE invitationId = ?`, invitationID)
	return groupIDs, err
}

// CleanupExpired 清理过期邀请
func (r *InvitationRepo) CleanupExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM invitations WHERE expiresAt < ?`, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

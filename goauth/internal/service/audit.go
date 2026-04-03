package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"goauth/internal/model"
)

// AuditService 审计服务
type AuditService struct {
	db *sqlx.DB
}

// NewAuditService 创建审计服务
func NewAuditService(db *sqlx.DB) *AuditService {
	return &AuditService{db: db}
}

// Log 记录审计日志
func (s *AuditService) Log(ctx context.Context, action string, actorID *string, targetID *string, details interface{}, ip string) error {
	var detailsJSON string
	if details != nil {
		b, err := json.Marshal(details)
		if err != nil {
			detailsJSON = "{}"
		} else {
			detailsJSON = string(b)
		}
	} else {
		detailsJSON = "{}"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_log (id, action, actorId, targetId, details, ip, createdAt)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, uuid.NewString(), action, actorID, targetID, detailsJSON, ip, time.Now())

	return err
}

// LogLogin 记录登录
func (s *AuditService) LogLogin(ctx context.Context, userID, ip string, success bool) error {
	action := model.AuditActionLogin
	if !success {
		action = "login_failed"
	}
	return s.Log(ctx, action, &userID, nil, map[string]bool{"success": success}, ip)
}

// LogLogout 记录登出
func (s *AuditService) LogLogout(ctx context.Context, userID, ip string) error {
	return s.Log(ctx, model.AuditActionLogout, &userID, nil, nil, ip)
}

// LogRegister 记录注册
func (s *AuditService) LogRegister(ctx context.Context, userID, ip string) error {
	return s.Log(ctx, model.AuditActionRegister, &userID, nil, nil, ip)
}

// LogUserApproved 记录用户审批
func (s *AuditService) LogUserApproved(ctx context.Context, actorID, targetID, ip string) error {
	return s.Log(ctx, model.AuditActionUserApproved, &actorID, &targetID, nil, ip)
}

// LogUserDisabled 记录用户禁用
func (s *AuditService) LogUserDisabled(ctx context.Context, actorID, targetID, ip string) error {
	return s.Log(ctx, model.AuditActionUserDisabled, &actorID, &targetID, nil, ip)
}

// List 列出审计日志
func (s *AuditService) List(ctx context.Context, limit, offset int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := s.db.SelectContext(ctx, &logs, `
		SELECT * FROM audit_log ORDER BY createdAt DESC LIMIT ? OFFSET ?
	`, limit, offset)
	return logs, err
}

// ListByActor 根据操作者列出审计日志
func (s *AuditService) ListByActor(ctx context.Context, actorID string, limit, offset int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := s.db.SelectContext(ctx, &logs, `
		SELECT * FROM audit_log WHERE actorId = ? ORDER BY createdAt DESC LIMIT ? OFFSET ?
	`, actorID, limit, offset)
	return logs, err
}

// ListByTarget 根据目标列出审计日志
func (s *AuditService) ListByTarget(ctx context.Context, targetID string, limit, offset int) ([]*model.AuditLog, error) {
	var logs []*model.AuditLog
	err := s.db.SelectContext(ctx, &logs, `
		SELECT * FROM audit_log WHERE targetId = ? ORDER BY createdAt DESC LIMIT ? OFFSET ?
	`, targetID, limit, offset)
	return logs, err
}

// CleanupOldLogs 清理旧的审计日志
func (s *AuditService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM audit_log WHERE createdAt < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

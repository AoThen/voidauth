package util

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// BruteForceProtector 暴力破解防护
type BruteForceProtector struct {
	db           *sqlx.DB
	maxAttempts  int
	blockMinutes int
}

// NewBruteForceProtector 创建暴力破解防护器
func NewBruteForceProtector(db *sqlx.DB, maxAttempts, blockMinutes int) *BruteForceProtector {
	return &BruteForceProtector{
		db:           db,
		maxAttempts:  maxAttempts,
		blockMinutes: blockMinutes,
	}
}

// RecordAttempt 记录登录尝试
func (p *BruteForceProtector) RecordAttempt(ctx context.Context, username, ip string, success bool) error {
	_, err := p.db.ExecContext(ctx, `
		INSERT INTO login_attempts (id, username, ip, success, createdAt)
		VALUES (?, ?, ?, ?, ?)
	`, uuid.NewString(), username, ip, success, time.Now())
	return err
}

// IsBlocked 检查是否被封锁
// 只封锁 IP，不封锁用户名，防止攻击者通过故意失败来封锁其他用户
func (p *BruteForceProtector) IsBlocked(ctx context.Context, username, ip string) (bool, error) {
	blockTime := time.Now().Add(-time.Duration(p.blockMinutes) * time.Minute)

	// 只检查 IP 封锁
	var ipAttempts int
	err := p.db.GetContext(ctx, &ipAttempts, `
		SELECT COUNT(*) FROM login_attempts
		WHERE ip = ? AND success = 0 AND createdAt > ?
	`, ip, blockTime)
	if err != nil {
		return false, err
	}

	return ipAttempts >= p.maxAttempts, nil
}

// ClearAttempts 清除登录尝试记录
func (p *BruteForceProtector) ClearAttempts(ctx context.Context, username string) error {
	_, err := p.db.ExecContext(ctx, `DELETE FROM login_attempts WHERE username = ?`, username)
	return err
}

// CleanupOldAttempts 清理旧的登录尝试记录
func (p *BruteForceProtector) CleanupOldAttempts(ctx context.Context, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	_, err := p.db.ExecContext(ctx, `DELETE FROM login_attempts WHERE createdAt < ?`, cutoff)
	return err
}

// GetRemainingAttempts 获取剩余尝试次数（基于 IP）
func (p *BruteForceProtector) GetRemainingAttempts(ctx context.Context, ip string) (int, error) {
	blockTime := time.Now().Add(-time.Duration(p.blockMinutes) * time.Minute)

	var count int
	err := p.db.GetContext(ctx, &count, `
		SELECT COUNT(*) FROM login_attempts
		WHERE ip = ? AND success = 0 AND createdAt > ?
	`, ip, blockTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return p.maxAttempts, nil
		}
		return 0, err
	}

	remaining := p.maxAttempts - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining, nil
}

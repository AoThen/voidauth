package util

import (
	"errors"

	"github.com/nbutton23/zxcvbn-go"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort   = errors.New("密码太短")
	ErrPasswordTooWeak    = errors.New("密码强度不足")
	ErrPasswordHashFailed = errors.New("密码哈希失败")
)

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", ErrPasswordHashFailed
	}
	return string(hash), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CheckPasswordStrength 检查密码强度
// minLen: 最小长度
// minScore: 最小强度分数 (0-4)
func CheckPasswordStrength(password string, minLen, minScore int) error {
	if len(password) < minLen {
		return ErrPasswordTooShort
	}

	result := zxcvbn.PasswordStrength(password, nil)
	if result.Score < minScore {
		return ErrPasswordTooWeak
	}

	return nil
}

// PasswordScore 返回密码强度分数 (0-4)
func PasswordScore(password string) int {
	result := zxcvbn.PasswordStrength(password, nil)
	return result.Score
}

// GenerateRandomPassword 生成随机密码
func GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	// Simple shuffle
	for i := range b {
		j := (i * 7 + 13) % len(b)
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

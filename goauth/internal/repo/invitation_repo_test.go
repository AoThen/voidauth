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

func setupInvitationTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS invitations (
		id TEXT PRIMARY KEY,
		email TEXT,
		username TEXT,
		name TEXT,
		emailVerified INTEGER DEFAULT 0,
		challenge TEXT NOT NULL UNIQUE,
		createdBy TEXT NOT NULL,
		expiresAt TEXT NOT NULL,
		createdAt TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS invitation_groups (
		invitationId TEXT NOT NULL,
		groupId TEXT NOT NULL,
		PRIMARY KEY (invitationId, groupId)
	);
	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		mfaRequired INTEGER DEFAULT 0,
		createdBy TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		updatedAt TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_invitations_challenge ON invitations(challenge);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func TestInvitationRepo_Create(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	email := "test@example.com"
	username := "testuser"
	invitation := &model.Invitation{
		Email:     &email,
		Username:  &username,
		Challenge: "test-challenge-123",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}

	err := repo.Create(ctx, invitation, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, invitation.ID, "ID should be generated")
}

func TestInvitationRepo_Create_WithGroups(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	// 创建分组
	now := model.Now()
	_, _ = db.ExecContext(ctx, "INSERT INTO groups (id, name, createdBy, createdAt, updatedAt) VALUES (?, ?, ?, ?, ?)",
		"group1", "Group 1", "admin", now, now)
	_, _ = db.ExecContext(ctx, "INSERT INTO groups (id, name, createdBy, createdAt, updatedAt) VALUES (?, ?, ?, ?, ?)",
		"group2", "Group 2", "admin", now, now)

	invitation := &model.Invitation{
		Challenge: "group-challenge",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}

	err := repo.Create(ctx, invitation, []string{"group1", "group2"})
	require.NoError(t, err)

	// 验证分组关联
	groupIDs, err := repo.GetInvitationGroups(ctx, invitation.ID)
	require.NoError(t, err)
	assert.Len(t, groupIDs, 2)
	assert.Contains(t, groupIDs, "group1")
	assert.Contains(t, groupIDs, "group2")
}

func TestInvitationRepo_FindByID(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	email := "find@example.com"
	invitation := &model.Invitation{
		Email:     &email,
		Challenge: "find-challenge",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}
	repo.Create(ctx, invitation, nil)

	found, err := repo.FindByID(ctx, invitation.ID)
	require.NoError(t, err)
	assert.Equal(t, invitation.ID, found.ID)
	assert.Equal(t, email, *found.Email)
}

func TestInvitationRepo_FindByID_NotFound(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	assert.Equal(t, ErrInvitationNotFound, err)
}

func TestInvitationRepo_FindByChallenge(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	challenge := "unique-challenge-123"
	email := "challenge@example.com"
	invitation := &model.Invitation{
		Email:     &email,
		Challenge: challenge,
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}
	repo.Create(ctx, invitation, nil)

	found, err := repo.FindByChallenge(ctx, challenge)
	require.NoError(t, err)
	assert.Equal(t, invitation.ID, found.ID)
	assert.Equal(t, email, *found.Email)
}

func TestInvitationRepo_FindByChallenge_Expired(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	challenge := "expired-challenge"
	invitation := &model.Invitation{
		Challenge: challenge,
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(-1 * time.Hour)}, // 已过期
	}
	repo.Create(ctx, invitation, nil)

	// FindByChallenge 现在返回邀请无论是否过期，过期检查由服务层负责
	found, err := repo.FindByChallenge(ctx, challenge)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, challenge, found.Challenge)
}

func TestInvitationRepo_FindByChallenge_NotFound(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	_, err := repo.FindByChallenge(ctx, "nonexistent-challenge")
	assert.Equal(t, ErrInvitationNotFound, err)
}

func TestInvitationRepo_Delete(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	invitation := &model.Invitation{
		Challenge: "delete-challenge",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}
	repo.Create(ctx, invitation, nil)

	err := repo.Delete(ctx, invitation.ID)
	require.NoError(t, err)

	// 验证已删除
	_, err = repo.FindByID(ctx, invitation.ID)
	assert.Equal(t, ErrInvitationNotFound, err)
}

func TestInvitationRepo_List(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	// 创建多个邀请
	for i := 0; i < 3; i++ {
		invitation := &model.Invitation{
			Challenge: "list-challenge-" + string(rune('0'+i)),
			CreatedBy: "admin-user",
			ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
		}
		repo.Create(ctx, invitation, nil)
	}

	invitations, err := repo.List(ctx)
	require.NoError(t, err)
	assert.Len(t, invitations, 3)
}

func TestInvitationRepo_CleanupExpired(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	// 创建过期邀请
	expiredInvitation := &model.Invitation{
		Challenge: "expired-invitation",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(-1 * time.Hour)},
	}
	repo.Create(ctx, expiredInvitation, nil)

	// 创建有效邀请
	validInvitation := &model.Invitation{
		Challenge: "valid-invitation",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}
	repo.Create(ctx, validInvitation, nil)

	// 清理过期邀请
	count, err := repo.CleanupExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// 验证有效邀请仍然存在
	_, err = repo.FindByID(ctx, validInvitation.ID)
	require.NoError(t, err)

	// 验证过期邀请已删除
	_, err = repo.FindByID(ctx, expiredInvitation.ID)
	assert.Equal(t, ErrInvitationNotFound, err)
}

func TestInvitationRepo_GetInvitationGroups(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	// 创建分组
	now := model.Now()
	_, _ = db.ExecContext(ctx, "INSERT INTO groups (id, name, createdBy, createdAt, updatedAt) VALUES (?, ?, ?, ?, ?)",
		"group-a", "Group A", "admin", now, now)

	invitation := &model.Invitation{
		Challenge: "groups-challenge",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}
	repo.Create(ctx, invitation, []string{"group-a"})

	groupIDs, err := repo.GetInvitationGroups(ctx, invitation.ID)
	require.NoError(t, err)
	assert.Len(t, groupIDs, 1)
	assert.Equal(t, "group-a", groupIDs[0])
}

func TestInvitationRepo_GetInvitationGroups_NoGroups(t *testing.T) {
	db := setupInvitationTestDB(t)
	repo := NewInvitationRepo(db)
	ctx := context.Background()

	invitation := &model.Invitation{
		Challenge: "no-groups-challenge",
		CreatedBy: "admin-user",
		ExpiresAt: model.CustomTime{Time: time.Now().Add(72 * time.Hour)},
	}
	repo.Create(ctx, invitation, nil)

	groupIDs, err := repo.GetInvitationGroups(ctx, invitation.ID)
	require.NoError(t, err)
	assert.Empty(t, groupIDs)
}

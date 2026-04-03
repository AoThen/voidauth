package service

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goauth/internal/model"
	"goauth/internal/repo"
)

func setupInvitationServiceTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	require.NoError(t, err)

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE,
		username TEXT UNIQUE NOT NULL,
		name TEXT,
		passwordHash TEXT,
		isAdmin INTEGER DEFAULT 0,
		emailVerified INTEGER DEFAULT 0,
		approved INTEGER DEFAULT 0,
		mfaRequired INTEGER DEFAULT 0,
		disabled INTEGER DEFAULT 0,
		createdAt TEXT NOT NULL,
		updatedAt TEXT NOT NULL
	);
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

func setupInvitationService(t *testing.T) (*sqlx.DB, *InvitationService) {
	t.Helper()
	db := setupInvitationServiceTestDB(t)

	invitationRepo := repo.NewInvitationRepo(db)
	groupRepo := repo.NewGroupRepo(db)
	invitationService := NewInvitationService(invitationRepo, groupRepo, db)

	return db, invitationService
}

func createTestUserForInvitation(t *testing.T, db *sqlx.DB, username string) *model.User {
	t.Helper()
	ctx := context.Background()

	user := &model.User{
		Username:      username,
		EmailVerified: true,
		Approved:      true,
		IsAdmin:       true,
	}

	err := repo.NewUserRepo(db).Create(ctx, user)
	require.NoError(t, err)

	return user
}

func createTestGroupForInvitation(t *testing.T, db *sqlx.DB, name, createdBy string) *model.Group {
	t.Helper()
	ctx := context.Background()

	group := &model.Group{
		Name:      name,
		CreatedBy: createdBy,
	}

	err := repo.NewGroupRepo(db).Create(ctx, group)
	require.NoError(t, err)

	return group
}

func TestInvitationService_CreateInvitation(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "invcreator")
	email := "invited@example.com"

	invitation, err := svc.CreateInvitation(
		ctx,
		&email,
		nil,
		nil,
		false,
		nil,
		user.ID,
		72,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, invitation.ID)
	assert.NotEmpty(t, invitation.Challenge)
	assert.Equal(t, email, *invitation.Email)
}

func TestInvitationService_CreateInvitation_WithGroups(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "groupinvcreator")
	group := createTestGroupForInvitation(t, db, "Invitation Group", user.ID)

	invitation, err := svc.CreateInvitation(
		ctx,
		nil,
		nil,
		nil,
		false,
		[]string{group.ID},
		user.ID,
		72,
	)

	require.NoError(t, err)

	// 验证分组关联
	groupIDs, err := svc.invitationRepo.GetInvitationGroups(ctx, invitation.ID)
	require.NoError(t, err)
	assert.Len(t, groupIDs, 1)
	assert.Equal(t, group.ID, groupIDs[0])
}

func TestInvitationService_CreateInvitation_DefaultExpiry(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "expiryuser")

	// 不提供过期时间，应该使用默认 72 小时
	invitation, err := svc.CreateInvitation(
		ctx,
		nil,
		nil,
		nil,
		false,
		nil,
		user.ID,
		0, // 使用默认值
	)

	require.NoError(t, err)
	// 验证过期时间约为 72 小时后
	expectedExpiry := time.Now().Add(72 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, invitation.ExpiresAt.Time, time.Minute)
}

func TestInvitationService_GetInvitationByChallenge(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "challengeuser")

	invitation, _ := svc.CreateInvitation(
		ctx,
		nil,
		nil,
		nil,
		false,
		nil,
		user.ID,
		72,
	)

	found, groupIDs, err := svc.GetInvitationByChallenge(ctx, invitation.Challenge)
	require.NoError(t, err)
	assert.Equal(t, invitation.ID, found.ID)
	assert.Empty(t, groupIDs)
}

func TestInvitationService_ValidateInvitation(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "validateuser")

	invitation, _ := svc.CreateInvitation(
		ctx,
		nil,
		nil,
		nil,
		false,
		nil,
		user.ID,
		72,
	)

	found, _, err := svc.ValidateInvitation(ctx, invitation.Challenge)
	require.NoError(t, err)
	assert.Equal(t, invitation.ID, found.ID)
}

func TestInvitationService_ValidateInvitation_Expired(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "expiredinvuser")

	// 创建过期邀请
	invitation := &model.Invitation{
		Challenge: "expired-challenge-test",
		CreatedBy: user.ID,
		ExpiresAt: model.CustomTime{Time: time.Now().Add(-1 * time.Hour)},
	}
	svc.invitationRepo.Create(ctx, invitation, nil)

	_, _, err := svc.ValidateInvitation(ctx, invitation.Challenge)
	assert.Equal(t, ErrInvitationExpired, err)
}

func TestInvitationService_ListInvitations(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "listinvuser")

	// 创建多个邀请
	for i := 0; i < 3; i++ {
		svc.CreateInvitation(
			ctx,
			nil,
			nil,
			nil,
			false,
			nil,
			user.ID,
			72,
		)
	}

	invitations, err := svc.ListInvitations(ctx)
	require.NoError(t, err)
	assert.Len(t, invitations, 3)
}

func TestInvitationService_DeleteInvitation(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "deleteinvuser")

	invitation, _ := svc.CreateInvitation(
		ctx,
		nil,
		nil,
		nil,
		false,
		nil,
		user.ID,
		72,
	)

	err := svc.DeleteInvitation(ctx, invitation.ID)
	require.NoError(t, err)

	// 验证删除
	invitations, _ := svc.ListInvitations(ctx)
	for _, inv := range invitations {
		assert.NotEqual(t, invitation.ID, inv.ID)
	}
}

func TestInvitationService_CleanupExpired(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "cleanupuser")

	// 创建过期邀请
	for i := 0; i < 2; i++ {
		invitation := &model.Invitation{
			Challenge: "expired-cleanup-" + string(rune('0'+i)),
			CreatedBy: user.ID,
			ExpiresAt: model.CustomTime{Time: time.Now().Add(-1 * time.Hour)},
		}
		svc.invitationRepo.Create(ctx, invitation, nil)
	}

	// 创建有效邀请
	svc.CreateInvitation(ctx, nil, nil, nil, false, nil, user.ID, 72)

	count, err := svc.CleanupExpired(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestInvitationService_CreateInvitation_WithEmailVerified(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "verifiedinvuser")

	invitation, err := svc.CreateInvitation(
		ctx,
		nil,
		nil,
		nil,
		true, // emailVerified
		nil,
		user.ID,
		72,
	)

	require.NoError(t, err)
	assert.True(t, invitation.EmailVerified)
}

func TestInvitationService_CreateInvitation_WithUsername(t *testing.T) {
	db, svc := setupInvitationService(t)
	ctx := context.Background()

	user := createTestUserForInvitation(t, db, "usernameinvuser")
	username := "predefined-username"

	invitation, err := svc.CreateInvitation(
		ctx,
		nil,
		&username,
		nil,
		false,
		nil,
		user.ID,
		72,
	)

	require.NoError(t, err)
	assert.Equal(t, username, *invitation.Username)
}

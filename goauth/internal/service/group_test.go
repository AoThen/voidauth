package service

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goauth/internal/model"
	"goauth/internal/repo"
)

func setupGroupTestDB(t *testing.T) *sqlx.DB {
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
	CREATE TABLE IF NOT EXISTS groups (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		mfaRequired INTEGER DEFAULT 0,
		createdBy TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		updatedAt TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS user_groups (
		userId TEXT NOT NULL,
		groupId TEXT NOT NULL,
		createdAt TEXT NOT NULL,
		PRIMARY KEY (userId, groupId),
		FOREIGN KEY (userId) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (groupId) REFERENCES groups(id) ON DELETE CASCADE
	);
	`
	_, err = db.Exec(schema)
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func setupGroupService(t *testing.T) (*sqlx.DB, *GroupService) {
	t.Helper()
	db := setupGroupTestDB(t)

	groupRepo := repo.NewGroupRepo(db)
	groupService := NewGroupService(groupRepo, db)

	return db, groupService
}

func createTestUserForGroup(t *testing.T, db *sqlx.DB, username string) *model.User {
	t.Helper()
	ctx := context.Background()

	user := &model.User{
		Username:      username,
		EmailVerified: true,
		Approved:      true,
	}

	err := repo.NewUserRepo(db).Create(ctx, user)
	require.NoError(t, err)

	return user
}

func createTestGroup(t *testing.T, db *sqlx.DB, name string, createdBy string) *model.Group {
	t.Helper()
	ctx := context.Background()

	group := &model.Group{
		Name:        name,
		MFARequired: false,
		CreatedBy:   createdBy,
	}

	err := repo.NewGroupRepo(db).Create(ctx, group)
	require.NoError(t, err)

	return group
}

func TestGroupService_Create(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "groupcreator")

	group := &model.Group{
		Name:        "Test Group",
		MFARequired: false,
		CreatedBy:   user.ID,
	}

	err := svc.Create(ctx, group)
	require.NoError(t, err)
	assert.NotEmpty(t, group.ID)
}

func TestGroupService_List(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "listuser")

	// 创建多个分组
	for i := 0; i < 3; i++ {
		createTestGroup(t, db, "Group "+string(rune('A'+i)), user.ID)
	}

	groups, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Len(t, groups, 3)
}

func TestGroupService_Update(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "updateuser")
	group := createTestGroup(t, db, "Update Test", user.ID)

	// 更新分组
	group.MFARequired = true
	err := svc.Update(ctx, group)
	require.NoError(t, err)

	// 验证更新
	groups, _ := svc.List(ctx)
	for _, g := range groups {
		if g.ID == group.ID {
			assert.True(t, g.MFARequired)
		}
	}
}

func TestGroupService_Delete(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "deleteuser")
	group := createTestGroup(t, db, "Delete Test", user.ID)

	err := svc.Delete(ctx, group.ID)
	require.NoError(t, err)

	// 验证删除
	groups, _ := svc.List(ctx)
	for _, g := range groups {
		assert.NotEqual(t, group.ID, g.ID)
	}
}

func TestGroupService_AddMember(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "memberuser")
	group := createTestGroup(t, db, "Member Test", user.ID)

	err := svc.AddMember(ctx, group.ID, user.ID)
	require.NoError(t, err)

	// 验证成员
	members, err := svc.GetMembers(ctx, group.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, user.ID, members[0])
}

func TestGroupService_RemoveMember(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "removemember")
	group := createTestGroup(t, db, "Remove Member Test", user.ID)

	// 添加成员
	svc.AddMember(ctx, group.ID, user.ID)

	// 移除成员
	err := svc.RemoveMember(ctx, group.ID, user.ID)
	require.NoError(t, err)

	// 验证成员已移除
	members, err := svc.GetMembers(ctx, group.ID)
	require.NoError(t, err)
	assert.Empty(t, members)
}

func TestGroupService_GetUserGroups(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	user := createTestUserForGroup(t, db, "usergroups")

	// 创建多个分组并添加用户
	for i := 0; i < 2; i++ {
		group := createTestGroup(t, db, "User Group "+string(rune('A'+i)), user.ID)
		svc.AddMember(ctx, group.ID, user.ID)
	}

	// 获取用户的分组
	groups, err := svc.GetUserGroups(ctx, user.ID)
	require.NoError(t, err)
	assert.Len(t, groups, 2)
}

func TestGroupService_GetMembers(t *testing.T) {
	db, svc := setupGroupService(t)
	ctx := context.Background()

	// 创建分组
	creator := createTestUserForGroup(t, db, "creator")
	group := createTestGroup(t, db, "Members Test", creator.ID)

	// 创建多个用户并添加到分组
	for i := 0; i < 3; i++ {
		user := createTestUserForGroup(t, db, "member"+string(rune('0'+i)))
		svc.AddMember(ctx, group.ID, user.ID)
	}

	members, err := svc.GetMembers(ctx, group.ID)
	require.NoError(t, err)
	assert.Len(t, members, 3)
}

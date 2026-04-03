package repo

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"goauth/internal/model"
)

func setupGroupTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 创建必要的表
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
		updatedAt TEXT NOT NULL,
		FOREIGN KEY (createdBy) REFERENCES users(id)
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
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func createTestUserForGroup(db *sqlx.DB, ctx context.Context, id, username string) error {
	now := time.Now()
	_, err := db.ExecContext(ctx, `
		INSERT INTO users (id, username, createdAt, updatedAt)
		VALUES (?, ?, ?, ?)
	`, id, username, now, now)
	return err
}

func TestGroupRepo_Create(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	// 先创建用户
	createTestUserForGroup(db, ctx, "user1", "testuser")

	group := &model.Group{
		Name:        "test-group",
		MFARequired: false,
		CreatedBy:   "user1",
	}

	err := repo.Create(ctx, group)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if group.ID == "" {
		t.Error("Create() did not set group ID")
	}
}

func TestGroupRepo_FindByID(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")

	group := &model.Group{Name: "test-group", CreatedBy: "user1"}
	repo.Create(ctx, group)

	// 验证记录存在
	var name string
	err := db.QueryRowContext(ctx, `SELECT name FROM groups WHERE id = ?`, group.ID).Scan(&name)
	if err != nil {
		t.Fatalf("Failed to query group: %v", err)
	}
	if name != group.Name {
		t.Errorf("FindByID() name = %s, want %s", name, group.Name)
	}
}

func TestGroupRepo_FindByID_NotFound(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	if err != ErrGroupNotFound {
		t.Errorf("FindByID() error = %v, want %v", err, ErrGroupNotFound)
	}
}

func TestGroupRepo_FindByName(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")

	group := &model.Group{Name: "test-group", CreatedBy: "user1"}
	repo.Create(ctx, group)

	// 验证可以通过名称查找
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM groups WHERE name = ?`, "test-group").Scan(&id)
	if err != nil {
		t.Fatalf("Failed to query group by name: %v", err)
	}
	if id != group.ID {
		t.Errorf("FindByName() ID = %s, want %s", id, group.ID)
	}
}

func TestGroupRepo_FindByName_NotFound(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	_, err := repo.FindByName(ctx, "nonexistent")
	if err != ErrGroupNotFound {
		t.Errorf("FindByName() error = %v, want %v", err, ErrGroupNotFound)
	}
}

func TestGroupRepo_Update(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")

	group := &model.Group{Name: "test-group", CreatedBy: "user1"}
	repo.Create(ctx, group)

	// 更新分组
	group.Name = "updated-group"
	group.MFARequired = true

	err := repo.Update(ctx, group)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 验证更新
	var name string
	var mfaRequired bool
	err = db.QueryRowContext(ctx, `SELECT name, mfaRequired FROM groups WHERE id = ?`, group.ID).Scan(&name, &mfaRequired)
	if err != nil {
		t.Fatalf("Failed to query group: %v", err)
	}
	if name != "updated-group" {
		t.Errorf("Update() name = %s, want updated-group", name)
	}
	if !mfaRequired {
		t.Error("Update() mfaRequired = false, want true")
	}
}

func TestGroupRepo_Delete(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")

	group := &model.Group{Name: "test-group", CreatedBy: "user1"}
	repo.Create(ctx, group)

	err := repo.Delete(ctx, group.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.FindByID(ctx, group.ID)
	if err != ErrGroupNotFound {
		t.Errorf("Delete() group still exists, error = %v", err)
	}
}

func TestGroupRepo_List(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")

	// 创建多个分组
	for _, name := range []string{"group-c", "group-a", "group-b"} {
		group := &model.Group{Name: name, CreatedBy: "user1"}
		repo.Create(ctx, group)
	}

	// 验证数量
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM groups`)
	if err != nil {
		t.Fatalf("Failed to count groups: %v", err)
	}
	if count != 3 {
		t.Errorf("List() count = %d, want 3", count)
	}

	// 验证按名称排序
	var names []string
	err = db.SelectContext(ctx, &names, `SELECT name FROM groups ORDER BY name`)
	if err != nil {
		t.Fatalf("Failed to query group names: %v", err)
	}
	if names[0] != "group-a" || names[1] != "group-b" || names[2] != "group-c" {
		t.Errorf("List() order = %v, want [group-a, group-b, group-c]", names)
	}
}

func TestGroupRepo_AddUserToGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")
	createTestUserForGroup(db, ctx, "user2", "testuser2")

	group := &model.Group{Name: "test-group", CreatedBy: "user1"}
	repo.Create(ctx, group)

	err := repo.AddUserToGroup(ctx, "user2", group.ID)
	if err != nil {
		t.Fatalf("AddUserToGroup() error = %v", err)
	}

	// 验证成员
	members, _ := repo.GetGroupMembers(ctx, group.ID)
	if len(members) != 1 || members[0] != "user2" {
		t.Errorf("GetGroupMembers() = %v, want [user2]", members)
	}
}

func TestGroupRepo_RemoveUserFromGroup(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")
	createTestUserForGroup(db, ctx, "user2", "testuser2")

	group := &model.Group{Name: "test-group", CreatedBy: "user1"}
	repo.Create(ctx, group)
	repo.AddUserToGroup(ctx, "user2", group.ID)

	err := repo.RemoveUserFromGroup(ctx, "user2", group.ID)
	if err != nil {
		t.Fatalf("RemoveUserFromGroup() error = %v", err)
	}

	members, _ := repo.GetGroupMembers(ctx, group.ID)
	if len(members) != 0 {
		t.Errorf("GetGroupMembers() = %v, want []", members)
	}
}

func TestGroupRepo_GetUserGroups(t *testing.T) {
	db := setupGroupTestDB(t)
	repo := NewGroupRepo(db)
	ctx := context.Background()

	createTestUserForGroup(db, ctx, "user1", "testuser")
	createTestUserForGroup(db, ctx, "user2", "testuser2")

	// 创建多个分组
	group1 := &model.Group{Name: "group1", CreatedBy: "user1"}
	group2 := &model.Group{Name: "group2", CreatedBy: "user1"}
	repo.Create(ctx, group1)
	repo.Create(ctx, group2)

	// 将 user2 加入 group1
	repo.AddUserToGroup(ctx, "user2", group1.ID)

	groups, err := repo.GetUserGroups(ctx, "user2")
	if err != nil {
		t.Fatalf("GetUserGroups() error = %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("GetUserGroups() returned %d groups, want 1", len(groups))
	}
	if groups[0].Name != "group1" {
		t.Errorf("GetUserGroups() name = %s, want group1", groups[0].Name)
	}
}

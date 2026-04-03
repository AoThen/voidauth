package repo

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"goauth/internal/model"
)

func setupUserTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 创建 users 表
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

func TestUserRepo_Create(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	email := "test@example.com"
	name := "Test User"
	passwordHash := "hashed_password"

	user := &model.User{
		Username:      "testuser",
		Email:         &email,
		Name:          &name,
		PasswordHash:  &passwordHash,
		IsAdmin:       false,
		EmailVerified: false,
		Approved:      true,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.ID == "" {
		t.Error("Create() did not set user ID")
	}

	// 验证可以找到用户
	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Username != user.Username {
		t.Errorf("FindByID() username = %s, want %s", found.Username, user.Username)
	}
}

func TestUserRepo_FindByID_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	_, err := repo.FindByID(ctx, "nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("FindByID() error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestUserRepo_FindByUsername(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Approved: true}
	repo.Create(ctx, user)

	found, err := repo.FindByUsername(ctx, "testuser")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("FindByUsername() ID = %s, want %s", found.ID, user.ID)
	}
}

func TestUserRepo_FindByUsername_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	_, err := repo.FindByUsername(ctx, "nonexistent")
	if err != ErrUserNotFound {
		t.Errorf("FindByUsername() error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestUserRepo_FindByEmail(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	email := "test@example.com"
	user := &model.User{Username: "testuser", Email: &email, Approved: true}
	repo.Create(ctx, user)

	found, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("FindByEmail() ID = %s, want %s", found.ID, user.ID)
	}
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "nonexistent@example.com")
	if err != ErrUserNotFound {
		t.Errorf("FindByEmail() error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestUserRepo_FindByInput(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	email := "test@example.com"
	user := &model.User{Username: "testuser", Email: &email, Approved: true}
	repo.Create(ctx, user)

	// 用用户名查找
	found, err := repo.FindByInput(ctx, "testuser")
	if err != nil {
		t.Fatalf("FindByInput(username) error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("FindByInput(username) ID = %s, want %s", found.ID, user.ID)
	}

	// 用邮箱查找
	found, err = repo.FindByInput(ctx, email)
	if err != nil {
		t.Fatalf("FindByInput(email) error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("FindByInput(email) ID = %s, want %s", found.ID, user.ID)
	}
}

func TestUserRepo_Update(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Approved: true}
	repo.Create(ctx, user)

	// 更新用户
	newName := "Updated Name"
	newEmail := "updated@example.com"
	user.Name = &newName
	user.Email = &newEmail
	user.IsAdmin = true

	err := repo.Update(ctx, user)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 验证更新
	found, _ := repo.FindByID(ctx, user.ID)
	if *found.Name != newName {
		t.Errorf("Update() name = %s, want %s", *found.Name, newName)
	}
	if *found.Email != newEmail {
		t.Errorf("Update() email = %s, want %s", *found.Email, newEmail)
	}
	if !found.IsAdmin {
		t.Error("Update() isAdmin = false, want true")
	}
}

func TestUserRepo_Delete(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	user := &model.User{Username: "testuser", Approved: true}
	repo.Create(ctx, user)

	err := repo.Delete(ctx, user.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 验证删除
	_, err = repo.FindByID(ctx, user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Delete() user still exists, error = %v", err)
	}
}

func TestUserRepo_List(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	// 创建多个用户
	for i := 0; i < 5; i++ {
		user := &model.User{Username: "user" + string(rune('0'+i)), Approved: true}
		repo.Create(ctx, user)
		time.Sleep(time.Millisecond) // 确保时间顺序
	}

	users, err := repo.List(ctx, 3, 0)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 3 {
		t.Errorf("List() returned %d users, want 3", len(users))
	}

	users, err = repo.List(ctx, 3, 3)
	if err != nil {
		t.Fatalf("List(offset) error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("List(offset) returned %d users, want 2", len(users))
	}
}

func TestUserRepo_Count(t *testing.T) {
	db := setupUserTestDB(t)
	repo := NewUserRepo(db)
	ctx := context.Background()

	// 初始计数
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}

	// 创建用户
	for i := 0; i < 3; i++ {
		user := &model.User{Username: "user" + string(rune('0'+i)), Approved: i < 2}
		repo.Create(ctx, user)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}

	approvedCount, err := repo.CountApproved(ctx)
	if err != nil {
		t.Fatalf("CountApproved() error = %v", err)
	}
	if approvedCount != 2 {
		t.Errorf("CountApproved() = %d, want 2", approvedCount)
	}
}

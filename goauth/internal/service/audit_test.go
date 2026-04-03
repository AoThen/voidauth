package service

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"goauth/internal/model"
)

func setupAuditTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite3", ":memory:?_foreign_keys=on")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// 创建审计日志表
	schema := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id TEXT PRIMARY KEY,
		action TEXT NOT NULL,
		actorId TEXT,
		targetId TEXT,
		details TEXT,
		ip TEXT NOT NULL,
		createdAt TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);
	CREATE INDEX IF NOT EXISTS idx_audit_log_createdAt ON audit_log(createdAt);
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

func TestAuditService_Log(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	actorID := "user1"
	targetID := "user2"
	details := map[string]string{"key": "value"}

	err := svc.Log(ctx, model.AuditActionLogin, &actorID, &targetID, details, "192.168.1.1")
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}

	// 验证记录
	var count int
	err = db.GetContext(ctx, &count, `SELECT COUNT(*) FROM audit_log WHERE action = ?`, model.AuditActionLogin)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}
	if count != 1 {
		t.Errorf("Log() did not create record, count = %d", count)
	}
}

func TestAuditService_LogLogin(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	err := svc.LogLogin(ctx, "user1", "192.168.1.1", true)
	if err != nil {
		t.Fatalf("LogLogin() error = %v", err)
	}

	var action string
	err = db.GetContext(ctx, &action, `SELECT action FROM audit_log WHERE actorId = ?`, "user1")
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if action != model.AuditActionLogin {
		t.Errorf("LogLogin() action = %s, want %s", action, model.AuditActionLogin)
	}

	// 测试失败登录
	err = svc.LogLogin(ctx, "user2", "192.168.1.2", false)
	if err != nil {
		t.Fatalf("LogLogin(false) error = %v", err)
	}

	err = db.GetContext(ctx, &action, `SELECT action FROM audit_log WHERE actorId = ?`, "user2")
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if action != "login_failed" {
		t.Errorf("LogLogin(false) action = %s, want login_failed", action)
	}
}

func TestAuditService_LogLogout(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	err := svc.LogLogout(ctx, "user1", "192.168.1.1")
	if err != nil {
		t.Fatalf("LogLogout() error = %v", err)
	}

	var action string
	err = db.GetContext(ctx, &action, `SELECT action FROM audit_log WHERE actorId = ?`, "user1")
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if action != model.AuditActionLogout {
		t.Errorf("LogLogout() action = %s, want %s", action, model.AuditActionLogout)
	}
}

func TestAuditService_LogRegister(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	err := svc.LogRegister(ctx, "user1", "192.168.1.1")
	if err != nil {
		t.Fatalf("LogRegister() error = %v", err)
	}

	var action string
	err = db.GetContext(ctx, &action, `SELECT action FROM audit_log WHERE actorId = ?`, "user1")
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if action != model.AuditActionRegister {
		t.Errorf("LogRegister() action = %s, want %s", action, model.AuditActionRegister)
	}
}

func TestAuditService_LogUserApproved(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	err := svc.LogUserApproved(ctx, "admin1", "user1", "192.168.1.1")
	if err != nil {
		t.Fatalf("LogUserApproved() error = %v", err)
	}

	var action, actorId string
	err = db.QueryRowContext(ctx, `SELECT action, actorId FROM audit_log WHERE targetId = ?`, "user1").Scan(&action, &actorId)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if action != model.AuditActionUserApproved {
		t.Errorf("LogUserApproved() action = %s, want %s", action, model.AuditActionUserApproved)
	}
	if actorId != "admin1" {
		t.Errorf("LogUserApproved() actorId = %s, want admin1", actorId)
	}
}

func TestAuditService_LogUserDisabled(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	err := svc.LogUserDisabled(ctx, "admin1", "user1", "192.168.1.1")
	if err != nil {
		t.Fatalf("LogUserDisabled() error = %v", err)
	}

	var action, actorId string
	err = db.QueryRowContext(ctx, `SELECT action, actorId FROM audit_log WHERE targetId = ?`, "user1").Scan(&action, &actorId)
	if err != nil {
		t.Fatalf("Failed to get record: %v", err)
	}
	if action != model.AuditActionUserDisabled {
		t.Errorf("LogUserDisabled() action = %s, want %s", action, model.AuditActionUserDisabled)
	}
}

func TestAuditService_List(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	// 插入多条记录
	for i := 0; i < 5; i++ {
		svc.Log(ctx, "action"+string(rune('0'+i)), nil, nil, nil, "192.168.1.1")
		time.Sleep(time.Millisecond) // 确保时间顺序
	}

	// 验证记录数量
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM audit_log`)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}
	if count != 5 {
		t.Errorf("Expected 5 records, got %d", count)
	}
}

func TestAuditService_ListByActor(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	actor1 := "actor1"
	actor2 := "actor2"

	svc.Log(ctx, "action1", &actor1, nil, nil, "192.168.1.1")
	svc.Log(ctx, "action2", &actor2, nil, nil, "192.168.1.1")
	svc.Log(ctx, "action3", &actor1, nil, nil, "192.168.1.1")

	// 验证 actor1 的记录数
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM audit_log WHERE actorId = ?`, actor1)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}
	if count != 2 {
		t.Errorf("actor1 should have 2 records, got %d", count)
	}
}

func TestAuditService_ListByTarget(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db)
	ctx := context.Background()

	target1 := "target1"
	target2 := "target2"

	svc.Log(ctx, "action1", nil, &target1, nil, "192.168.1.1")
	svc.Log(ctx, "action2", nil, &target2, nil, "192.168.1.1")
	svc.Log(ctx, "action3", nil, &target1, nil, "192.168.1.1")

	// 验证 target1 的记录数
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM audit_log WHERE targetId = ?`, target1)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}
	if count != 2 {
		t.Errorf("target1 should have 2 records, got %d", count)
	}
}

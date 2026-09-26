package db

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testAccount 与 testAccountExpanded 共用同一张表，用于验证 AutoMigrate 的结构增量。
type testAccount struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func (testAccount) TableName() string { return "test_accounts" }

type testAccountExpanded struct {
	ID    uint `gorm:"primaryKey"`
	Name  string
	Email string
}

func (testAccountExpanded) TableName() string { return "test_accounts" }

// setupTestDB 用临时文件数据库替换全局实例，并保证测试后恢复。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "database.db")
	g, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	prevDB, prevPath := db, dbPath
	SetDB(g)
	dbPath = path
	t.Cleanup(func() {
		db, dbPath = prevDB, prevPath
	})
	return g
}

// TestMigrateAppliesPendingOnce 验证待执行迁移只执行一次并记录版本。
func TestMigrateAppliesPendingOnce(t *testing.T) {
	setupTestDB(t)
	runs := 0
	migrations := []Migration{{
		Version: 1,
		Name:    "create-test-account",
		Up: func(tx *gorm.DB) error {
			runs++
			return tx.AutoMigrate(&testAccount{})
		},
	}}

	if err := Migrate(Schema{Migrations: migrations}); err != nil {
		t.Fatalf("首次迁移失败: %v", err)
	}
	if err := Migrate(Schema{Migrations: migrations}); err != nil {
		t.Fatalf("重复迁移失败: %v", err)
	}
	if runs != 1 {
		t.Fatalf("迁移应仅执行一次，实际 %d 次", runs)
	}
	if !db.Migrator().HasTable(&testAccount{}) {
		t.Fatal("迁移未创建数据表")
	}
}

// TestMigrateAutoMigratesModels 验证 Models 以 AutoMigrate 方式补齐新增字段。
func TestMigrateAutoMigratesModels(t *testing.T) {
	g := setupTestDB(t)
	if err := g.AutoMigrate(&testAccount{}); err != nil {
		t.Fatalf("初始化基础表失败: %v", err)
	}

	if err := Migrate(Schema{Models: []interface{}{&testAccountExpanded{}}}); err != nil {
		t.Fatalf("结构同步失败: %v", err)
	}
	if !g.Migrator().HasColumn(&testAccountExpanded{}, "Email") {
		t.Fatal("新增字段未被自动迁移")
	}
}

// TestMigrateStopsOnFailureAndRetries 验证迁移失败不记录版本，修复后可重试。
func TestMigrateStopsOnFailureAndRetries(t *testing.T) {
	setupTestDB(t)
	boom := errors.New("boom")
	migrations := []Migration{
		{Version: 1, Name: "ok", Up: func(tx *gorm.DB) error { return nil }},
		{Version: 2, Name: "bad", Up: func(tx *gorm.DB) error { return boom }},
	}

	if err := Migrate(Schema{Migrations: migrations}); !errors.Is(err, boom) {
		t.Fatalf("迁移失败应返回原始原因，实际 %v", err)
	}
	assertAppliedVersions(t, 1)

	migrations[1].Up = func(tx *gorm.DB) error { return nil }
	if err := Migrate(Schema{Migrations: migrations}); err != nil {
		t.Fatalf("修复后重试失败: %v", err)
	}
	assertAppliedVersions(t, 1, 2)
}

func assertAppliedVersions(t *testing.T, want ...int) {
	t.Helper()
	var rows []schemaMigration
	if err := db.Find(&rows).Error; err != nil {
		t.Fatalf("读取版本记录失败: %v", err)
	}
	if len(rows) != len(want) {
		t.Fatalf("版本记录数量应为 %d，实际 %d: %+v", len(want), len(rows), rows)
	}
	got := map[int]bool{}
	for _, r := range rows {
		got[r.Version] = true
	}
	for _, v := range want {
		if !got[v] {
			t.Fatalf("缺少版本记录 v%d: %+v", v, rows)
		}
	}
}

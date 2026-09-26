package db

import (
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

// Migration 描述一次版本化的数据库变更。
// Version 必须为从 1 开始的正整数；Name 用于日志与排错；Up 在独立事务内执行。
// 已发布并应用的条目不得再修改其语义，只能追加新版本。
type Migration struct {
	Version int
	Name    string
	Up      func(tx *gorm.DB) error
}

// Schema 描述数据库的结构来源与版本化迁移。
type Schema struct {
	// Models 交给 GORM AutoMigrate，处理新增表 / 新增列 / 新增索引等安全增量。
	Models []interface{}
	// Migrations 处理 AutoMigrate 无法安全完成的变更：
	// 数据回填、字段改名、字段改类型、删除列等。
	Migrations []Migration
}

// schemaMigration 记录已应用的迁移版本，是版本化迁移的唯一状态来源。
type schemaMigration struct {
	Version   int       `gorm:"primaryKey" json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"appliedAt"`
}

// Migrate 同步数据库结构并应用尚未执行的版本化迁移。
//
// 执行顺序：先 AutoMigrate 同步 Schema.Models（使新增列对回填类迁移可见），
// 再按 Version 升序执行 Schema.Migrations。每个版本化迁移在独立事务内完成并写入
// 版本记录；失败即中止（已成功的事务保留，失败事务整体回滚，下次启动自动重试）。
func Migrate(s Schema) error {
	if db == nil {
		return nil
	}
	if len(s.Models) > 0 {
		if err := db.AutoMigrate(s.Models...); err != nil {
			return fmt.Errorf("同步数据库结构失败: %w", err)
		}
	}
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("创建迁移版本表失败: %w", err)
	}
	ordered, err := normalizeMigrations(s.Migrations)
	if err != nil {
		return err
	}
	applied, err := appliedVersions()
	if err != nil {
		return fmt.Errorf("读取迁移版本失败: %w", err)
	}
	for _, m := range ordered {
		if applied[m.Version] {
			continue
		}
		if err := applyMigration(m); err != nil {
			return fmt.Errorf("执行迁移 v%d(%s) 失败: %w", m.Version, m.Name, err)
		}
	}
	return nil
}

// normalizeMigrations 复制并按 Version 升序排序，校验版本为正整数且不重复。
func normalizeMigrations(migrations []Migration) ([]Migration, error) {
	ordered := make([]Migration, len(migrations))
	copy(ordered, migrations)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Version < ordered[j].Version })
	seen := make(map[int]struct{}, len(ordered))
	for _, m := range ordered {
		if m.Version <= 0 {
			return nil, fmt.Errorf("迁移版本必须为正整数: %d(%s)", m.Version, m.Name)
		}
		if _, ok := seen[m.Version]; ok {
			return nil, fmt.Errorf("迁移版本重复: v%d", m.Version)
		}
		seen[m.Version] = struct{}{}
	}
	return ordered, nil
}

// appliedVersions 返回已应用的迁移版本集合。
func appliedVersions() (map[int]bool, error) {
	var rows []schemaMigration
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	applied := make(map[int]bool, len(rows))
	for _, r := range rows {
		applied[r.Version] = true
	}
	return applied, nil
}

// applyMigration 在事务内执行单个迁移并写入版本记录。
func applyMigration(m Migration) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if m.Up != nil {
			if err := m.Up(tx); err != nil {
				return err
			}
		}
		return tx.Create(&schemaMigration{Version: m.Version, Name: m.Name, AppliedAt: time.Now()}).Error
	})
}

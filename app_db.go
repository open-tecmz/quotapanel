package main

import (
	"fmt"

	"quotapanel/backend/base/db"
	"quotapanel/backend/base/logging"
	"quotapanel/backend/quota"
)

// initDatabase 初始化数据库：打开连接、迁移前滚动备份、同步结构并执行版本化迁移。
//
// 备份在迁移前进行，保证任何结构/数据变更都可回退到变更前状态；
// db.Migrate 内部先用 AutoMigrate 同步 quota.Account（新增列 / 表 / 索引），
// 再执行 quota.Migrations 中的版本化迁移（数据回填、改名、改类型等）。
func (a *App) initDatabase() {
	if err := db.Init(a.dataDir); err != nil {
		logging.Error("初始化数据库失败: %v", err)
		a.ReportError(fmt.Sprintf("db.Init failed: %v", err), "", "startup/db")
		return
	}

	if path, err := db.Backup(); err != nil {
		logging.Error("数据库备份失败: %v", err)
	} else if path != "" {
		logging.Info("数据库已备份：%s", path)
	}

	if err := db.Migrate(db.Schema{
		Models:     []interface{}{&quota.Account{}},
		Migrations: quota.Migrations(),
	}); err != nil {
		logging.Error("数据库迁移失败: %v", err)
		a.ReportError(fmt.Sprintf("db.Migrate failed: %v", err), "", "startup/db")
	}
}

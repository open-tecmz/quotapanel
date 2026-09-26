package quota

import "quotapanel/backend/base/db"

// Migrations 返回额度模块的版本化数据库迁移。
//
// 仅在 GORM AutoMigrate 无法安全完成的变更时新增条目：
// 数据回填、字段改名、字段改类型、删除列等。
// Version 必须从 1 开始连续递增，已发布的条目不得修改语义，只能追加新版本。
func Migrations() []db.Migration {
	return []db.Migration{}
}

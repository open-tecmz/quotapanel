package db

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db     *gorm.DB
	dbPath string
	once   sync.Once
)

// DB returns the singleton database instance.
// It must be initialized via Init before first use.
func DB() *gorm.DB {
	return db
}

// Init initializes the SQLite database at <dataDir>/db/database.db.
// It creates the directory if it does not exist, then opens the database.
// Structure synchronization and versioned migrations are handled separately
// by Migrate (see migrate.go).
func Init(dataDir string) error {
	var initErr error
	once.Do(func() {
		dbDir := filepath.Join(dataDir, "db")
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			initErr = err
			return
		}
		dbPath = filepath.Join(dbDir, "database.db")
		gormDB, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			initErr = err
			return
		}
		db = gormDB
	})
	return initErr
}

// Close closes the underlying database connection.
func Close() {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.Close()
	}
}

// SetDB 替换全局 DB 实例，主要用于测试注入内存数据库。
func SetDB(gormDB *gorm.DB) {
	db = gormDB
}

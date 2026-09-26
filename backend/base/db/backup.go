package db

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	backupDirName  = "backup"
	backupPrefix   = "database-"
	backupSuffix   = ".db"
	maxBackupFiles = 5
)

// Backup 在迁移开始前对当前数据库做一次滚动备份，返回新备份文件路径。
// 以下情况不创建备份并返回空字符串：数据库尚未初始化、文件为空（全新库）、
// 或数据库自最近一次备份以来未发生变化。
// 备份写入 <dataDir>/db/backup/，最多保留 maxBackupFiles 份。
func Backup() (string, error) {
	if dbPath == "" {
		return "", nil
	}
	return backupFile(dbPath, maxBackupFiles)
}

// backupFile 将 path 复制到同目录下的 backup/ 目录，并滚动清理历史备份。
func backupFile(path string, keep int) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if info.Size() == 0 {
		return "", nil
	}

	// 尽最大努力将 WAL 内容合并回主库，确保复制出的文件是完整快照。
	if db != nil {
		_ = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
	}

	dir := filepath.Join(filepath.Dir(path), backupDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// 数据库自最近一次备份以来没有变化，无需重复备份。
	latest, err := latestBackupModTime(dir)
	if err != nil {
		return "", err
	}
	if !latest.IsZero() && !info.ModTime().After(latest) {
		return "", nil
	}

	dest := uniqueBackupPath(dir)
	if err := copyFile(path, dest); err != nil {
		return "", err
	}
	if err := pruneBackups(dir, keep); err != nil {
		return "", err
	}
	return dest, nil
}

// uniqueBackupPath 生成 database-<时间戳>.db；同一秒内的冲突追加序号。
func uniqueBackupPath(dir string) string {
	base := filepath.Join(dir, backupPrefix+time.Now().Format("20060102-150405"))
	candidate := base + backupSuffix
	for i := 1; ; i++ {
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d%s", base, i, backupSuffix)
	}
}

// latestBackupModTime 返回备份目录中最新的备份修改时间；无备份时返回零值。
func latestBackupModTime(dir string) (time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	var latest time.Time
	for _, e := range entries {
		if e.IsDir() || !isBackupFile(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest, nil
}

// pruneBackups 仅保留最新的 keep 份备份（按时间戳命名，字典序即时间序）。
func pruneBackups(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && isBackupFile(e.Name()) {
			names = append(names, e.Name())
		}
	}
	if len(names) <= keep {
		return nil
	}
	sort.Strings(names)
	for _, name := range names[:len(names)-keep] {
		_ = os.Remove(filepath.Join(dir, name))
	}
	return nil
}

func isBackupFile(name string) bool {
	return strings.HasPrefix(name, backupPrefix) && strings.HasSuffix(name, backupSuffix)
}

// copyFile 复制 src 到 dst，失败时清理半成品文件。
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

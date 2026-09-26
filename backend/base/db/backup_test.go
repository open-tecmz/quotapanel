package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBackupCreatesSnapshotAndSkipsUnchanged 验证备份生成快照，且无变化时不重复备份。
func TestBackupCreatesSnapshotAndSkipsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database.db")
	if err := os.WriteFile(path, []byte("snapshot"), 0o644); err != nil {
		t.Fatal(err)
	}

	backup, err := backupFile(path, 3)
	if err != nil {
		t.Fatalf("备份失败: %v", err)
	}
	if backup == "" {
		t.Fatal("应生成备份文件")
	}
	data, err := os.ReadFile(backup)
	if err != nil || string(data) != "snapshot" {
		t.Fatalf("备份内容不正确: %q %v", data, err)
	}

	if again, err := backupFile(path, 3); err != nil || again != "" {
		t.Fatalf("数据库未变化时不应重复备份: %q %v", again, err)
	}
}

// TestBackupSkipsEmptyDatabase 验证全新空库不做无意义备份。
func TestBackupSkipsEmptyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database.db")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := backupFile(path, 3); err != nil || got != "" {
		t.Fatalf("空数据库不应备份: %q %v", got, err)
	}
}

// TestBackupPrunesOldFiles 验证滚动保留上限。
func TestBackupPrunesOldFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	// 每次把数据库修改时间前移，强制生成新备份。
	for i := 0; i < 4; i++ {
		future := time.Now().Add(time.Duration(i+1) * time.Minute)
		if err := os.Chtimes(path, future, future); err != nil {
			t.Fatal(err)
		}
		if _, err := backupFile(path, 2); err != nil {
			t.Fatalf("备份失败: %v", err)
		}
	}

	entries, err := os.ReadDir(filepath.Join(dir, backupDirName))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if isBackupFile(e.Name()) {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("应仅保留 2 份备份，实际 %d", count)
	}
}

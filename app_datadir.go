package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DataRootEnv 数据根目录环境变量。设置后应用数据（config.json、db/、study.json、system_logs/）
// 写入该目录，优先级最高，用于将正式使用数据与开发测试数据完全隔离。
const DataRootEnv = "QUOTAPANEL_DATA_ROOT"

func getAppDataDir() string {
	// 始终先加载客户端配置；开发/打包环境变量仍可覆盖最终数据目录。
	homeDir, _ := os.UserHomeDir()
	configured := readClientDataRoot(homeDir)
	if v := strings.TrimSpace(os.Getenv(DataRootEnv)); v != "" {
		return normalizeDataDir(v)
	}
	if configured != "" {
		return normalizeDataDir(configured)
	}
	return defaultDataDir()
}

// defaultDataDir 返回默认数据根目录（~/.quotapanel/data）。
func defaultDataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".quotapanel", "data")
}

// isDefaultDataDir 判断给定目录是否为默认数据根目录。
func isDefaultDataDir(dir string) bool {
	return filepath.Clean(dir) == filepath.Clean(defaultDataDir())
}

// readClientDataRoot 读取客户端配置；首次启动时写入默认 dataRoot。
func readClientDataRoot(homeDir string) string {
	defaultRoot := filepath.Join(homeDir, ".quotapanel", "data")
	configPath := filepath.Join(homeDir, ".quotapanel", "client.json")
	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		if os.MkdirAll(filepath.Dir(configPath), 0755) == nil {
			if encoded, encodeErr := json.MarshalIndent(map[string]string{"dataRoot": defaultRoot}, "", "  "); encodeErr == nil {
				_ = os.WriteFile(configPath, encoded, 0644)
			}
		}
		return defaultRoot
	}
	if err != nil {
		return defaultRoot
	}
	var cfg struct {
		DataRoot string `json:"dataRoot"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultRoot
	}
	if root := strings.TrimSpace(cfg.DataRoot); root != "" {
		return root
	}
	return defaultRoot
}

// normalizeDataDir 将 dataRoot 配置规范化为绝对路径（支持 ~ 展开）
func normalizeDataDir(p string) string {
	dir, err := filepath.Abs(expandHomeDir(p))
	if err != nil {
		return expandHomeDir(p)
	}
	return dir
}

// expandHomeDir 展开路径开头的 ~ 为用户主目录
func expandHomeDir(p string) string {
	homeDir, _ := os.UserHomeDir()
	if p == "~" {
		return homeDir
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(homeDir, p[2:])
	}
	return p
}

package quota

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"quotapanel/backend/base/db"
)

// sessionBrowser 是带会话状态的假浏览器，用于验证自动查询的跳过规则。
type sessionBrowser struct{ hasSession bool }

func (sessionBrowser) Open(uint, string) error                  { return nil }
func (sessionBrowser) Complete(uint)                            {}
func (sessionBrowser) Remove(uint) error                        { return nil }
func (sessionBrowser) Close()                                   {}
func (sessionBrowser) Run(uint, string, string) (string, error) { return "", nil }
func (s sessionBrowser) HasSession(uint) bool                   { return s.hasSession }

// setupQuotaDB 用临时文件数据库替换全局实例，并保证测试后恢复。
func setupQuotaDB(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "database.db")
	g, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	if err := g.AutoMigrate(&Account{}); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	prev := db.DB()
	db.SetDB(g)
	t.Cleanup(func() { db.SetDB(prev) })
}

// TestCanAutoQuery 验证示例账号与未登录的浏览器类账号不参与自动查询。
func TestCanAutoQuery(t *testing.T) {
	cases := []struct {
		name    string
		account Account
		browser BrowserRunner
		demos   []uint
		want    bool
	}{
		{"普通 API 账号", Account{ID: 1, Provider: "deepseek"}, nil, nil, true},
		{"示例账号", Account{ID: 2, Provider: "deepseek"}, nil, []uint{2}, false},
		{"未登录浏览器账号", Account{ID: 3, Provider: "claude-web"}, sessionBrowser{hasSession: false}, nil, false},
		{"已登录浏览器账号", Account{ID: 4, Provider: "claude-web"}, sessionBrowser{hasSession: true}, nil, true},
		{"无浏览器能力", Account{ID: 5, Provider: "claude-web"}, nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewService()
			s.SetBrowserRunner(c.browser)
			s.SetDemoAccounts(c.demos)
			if got := s.canAutoQuery(&c.account); got != c.want {
				t.Fatalf("canAutoQuery = %v, 期望 %v", got, c.want)
			}
		})
	}
}

// TestQueryAllSkipsNonAutoQueryAccounts 验证全量查询跳过示例账号与未登录浏览器账号。
func TestQueryAllSkipsNonAutoQueryAccounts(t *testing.T) {
	setupQuotaDB(t)

	demo := &Account{Provider: "claude-web", Name: "示例"}
	ready := &Account{Provider: "codex-web", Name: "已登录"}
	if err := db.DB().Create(demo).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.DB().Create(ready).Error; err != nil {
		t.Fatal(err)
	}

	s := NewService()
	s.SetBrowserRunner(sessionBrowser{hasSession: true})
	s.SetDemoAccounts([]uint{demo.ID})

	snaps, err := s.QueryAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 1 || snaps[0].AccountID != ready.ID {
		t.Fatalf("全量查询应只包含已登录账号 %d，实际 %+v", ready.ID, snaps)
	}
}

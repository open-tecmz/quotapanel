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

// TestAddAccountAppendsAndReorder 验证新账号追加在末尾、可自定义排序并持久化。
func TestAddAccountAppendsAndReorder(t *testing.T) {
	setupQuotaDB(t)
	s := NewService()

	first, err := s.AddAccount("deepseek", "第一个", "sk-1", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.AddAccount("deepseek", "第二个", "sk-2", "")
	if err != nil {
		t.Fatal(err)
	}

	list, err := s.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != first.ID || list[1].ID != second.ID {
		t.Fatalf("新账号应按添加顺序排列，实际 %+v", orderOf(list))
	}

	if err := s.ReorderAccounts([]uint{second.ID, first.ID}); err != nil {
		t.Fatal(err)
	}
	list, err = s.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("重排未生效，实际 %+v", orderOf(list))
	}
	if list[0].SortOrder != 1 || list[1].SortOrder != 2 {
		t.Fatalf("重排后的 sortOrder 应为 1、2，实际 %d、%d", list[0].SortOrder, list[1].SortOrder)
	}

	// 重排后新增的账号仍追加在末尾。
	third, err := s.AddAccount("deepseek", "第三个", "sk-3", "")
	if err != nil {
		t.Fatal(err)
	}
	list, err = s.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[2].ID != third.ID {
		t.Fatalf("新增账号应排在末尾，实际 %+v", orderOf(list))
	}
}

// TestSortOrderBackfillMigration 验证新增 sort_order 列后按 id 回填，保持原有展示顺序。
func TestSortOrderBackfillMigration(t *testing.T) {
	setupQuotaDB(t)

	// 模拟升级前的历史数据：sort_order 均为默认值 0。
	for _, name := range []string{"甲", "乙", "丙"} {
		if err := db.DB().Create(&Account{Provider: "deepseek", Name: name, Key: "sk"}).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := db.Migrate(db.Schema{Models: []interface{}{&Account{}}, Migrations: Migrations()}); err != nil {
		t.Fatalf("执行迁移失败: %v", err)
	}

	var accounts []Account
	if err := db.DB().Order("sort_order asc, id asc").Find(&accounts).Error; err != nil {
		t.Fatal(err)
	}
	for _, acc := range accounts {
		if acc.SortOrder != int(acc.ID) {
			t.Fatalf("账号 %d 的 sort_order 应回填为 id，实际 %d", acc.ID, acc.SortOrder)
		}
	}
}

// orderOf 提取账号 ID 顺序，便于断言失败时输出。
func orderOf(list []AccountView) []uint {
	ids := make([]uint, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	return ids
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

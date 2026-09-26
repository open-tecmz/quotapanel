package quota

import "time"

// QueryMode 表示额度查询的方式。
// api：直接调用供应商标注的 HTTP 接口；
// browser：需要通过浏览器界面（登录态 / 页面渲染）才能获取，依赖 BrowserRunner。
type QueryMode = string

const (
	ModeAPI     QueryMode = "api"
	ModeBrowser QueryMode = "browser"
)

// ProviderInfo 描述一个额度供应商标注的元信息，供前端展示与表单使用。
type ProviderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	KeyLabel    string `json:"keyLabel"`
	KeyHint     string `json:"keyHint"`
	DocsURL     string `json:"docsUrl"`
}

// Account 是一条已添加的订阅账号（持久化模型）。
// Key 为订阅凭证（API Key 或登录令牌），Note 为用户备注。
type Account struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Provider  string    `json:"provider"`
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AccountView 是返回给前端的账号数据（含完整 Key，供复制）。
type AccountView struct {
	ID           uint      `json:"id"`
	Provider     string    `json:"provider"`
	ProviderName string    `json:"providerName"`
	Name         string    `json:"name"`
	Note         string    `json:"note"`
	Key          string    `json:"key"`
	KeyMasked    string    `json:"keyMasked"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Window 表示一个用量窗口（滚动 / 每周 / 月度等）。
type Window struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Percent float64 `json:"percent"`
	Status  string  `json:"status"` // ok | warning | exceeded | rate-limited
	Detail  string  `json:"detail"` // 例如 "$1.16 / $14.00"
	ResetAt string  `json:"resetAt"`
}

// Stat 表示一条键值展示项（额度余额 / 用量汇总等）。
type Stat struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Muted bool   `json:"muted"`
}

// Snapshot 是各 provider 归一化后的额度快照，前端按统一结构渲染。
type Snapshot struct {
	AccountID   uint      `json:"accountId"`
	Provider    string    `json:"provider"`
	AccountName string    `json:"accountName"`
	Plan        string    `json:"plan"`
	Status      string    `json:"status"`
	Windows     []Window  `json:"windows"`
	Balances    []Stat    `json:"balances"`
	Stats       []Stat    `json:"stats"`
	Summary     string    `json:"summary"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Error       string    `json:"error,omitempty"`
}

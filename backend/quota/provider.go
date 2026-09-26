package quota

import (
	"sort"
	"time"
)

// formatISODisplay 将 ISO/RFC3339 时间字符串转换为本地可读时间（MM-DD HH:mm）。
func formatISODisplay(v string) string {
	if v == "" {
		return "-"
	}
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t.Local().Format("01-02 15:04")
		}
	}
	return v
}

// formatEpochMillis 将毫秒时间戳转换为本地可读时间（MM-DD HH:mm）。
func formatEpochMillis(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	return time.UnixMilli(ms).Local().Format("01-02 15:04")
}

// normalizeStatus 将 provider 的状态与用量百分比归一化为前端统一状态：
// ok | warning | exceeded。
func normalizeStatus(status string, percent float64) string {
	if status == "rate-limited" || percent >= 100 {
		return "exceeded"
	}
	if percent >= 80 {
		return "warning"
	}
	return "ok"
}

func now() time.Time { return time.Now() }

// truncateText 截断过长文本，用于日志与错误提示中展示原始响应片段。
func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// BrowserRunner 允许 provider 借助浏览器界面完成额度查询。
// 对于「只能通过界面获取」的订阅（例如必须网页登录后查看的账号），
// provider 声明 Mode = ModeBrowser，并在 Query 中通过该接口打开页面、执行脚本。
// 未注入时（例如纯后端环境）Browser 为 nil，provider 应返回可读的错误。
type BrowserRunner interface {
	Open(accountID uint, url string) error
	Complete(accountID uint)
	Remove(accountID uint) error
	Close()
	Run(accountID uint, url string, script string) (string, error)
}

// QueryContext 是 provider 查询额度时的上下文。
type QueryContext struct {
	Account *Account
	Browser BrowserRunner
}

// Provider 是所有额度供应商需要实现的统一接口。
// 新增一个供应商只需实现该接口并在 init 中调用 Register。
type Provider interface {
	Info() ProviderInfo
	Query(qc *QueryContext) (*Snapshot, error)
}

var registry = map[string]Provider{}

// Register 注册一个 provider（在 provider 文件的 init 中调用）。
func Register(p Provider) {
	registry[p.Info().ID] = p
}

// Providers 返回全部已注册的 provider，按 ID 排序。
func Providers() []Provider {
	list := make([]Provider, 0, len(registry))
	for _, p := range registry {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Info().ID < list[j].Info().ID })
	return list
}

// GetProvider 按 ID 查找 provider。
func GetProvider(id string) (Provider, bool) {
	p, ok := registry[id]
	return p, ok
}

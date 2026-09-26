package quota

import (
	"fmt"
	"strconv"
	"strings"
)

// 小米 MiMo Token Plan 的额度只在控制台页面上展示且需要登录，官方未开放用 API Key 查询额度
// （tp- Key 仅用于模型调用）。因此这里通过浏览器登录会话读取渲染后的 DOM。
const mimoPlatformURL = "https://platform.xiaomimimo.com/console/plan-manage"

func init() { Register(mimoProvider{}) }

type mimoProvider struct{}

func (mimoProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          "mimo",
		Name:        "Xiaomi MiMo Token Plan",
		Description: "在独立登录窗口登录小米 MiMo 平台后，读取 Token Plan 套餐用量与账户余额",
		Mode:        ModeBrowser,
		DocsURL:     "https://mimo.mi.com/docs/zh-CN/tokenplan/Token%20Plan/quick-access",
	}
}

func (mimoProvider) LoginURL() string { return mimoPlatformURL }

// 页面解析规则见 backend/quota/scrape-rules/mimo.json（与 scripts/webfetch 共用）。
// 平台改版时只需更新该规则文件的选择器，无需改动这里。

// mimoPageData 是页面 DOM 抽取出的结构化数据（字段名对应 scrape-rules/mimo.json）。
type mimoPageData struct {
	PlanName     string   `json:"planName"`
	ExpiresAt    string   `json:"expiresAt"`
	Used         string   `json:"used"`
	Limit        string   `json:"limit"`
	Percent      *float64 `json:"percent"`
	TokenTotal   string   `json:"tokenTotal"`
	RequestCount string   `json:"requestCount"`
}

func (mimoProvider) Query(qc *QueryContext) (*Snapshot, error) {
	script, err := buildScrapeRuleScript("mimo")
	if err != nil {
		return nil, err
	}
	var page mimoPageData
	if err := browserFetch(qc, mimoPlatformURL, script, &page); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "mimo", Plan: page.PlanName, Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	if snap.Plan == "" {
		snap.Plan = "Token Plan"
	}

	if page.Limit != "" || page.Used != "" {
		percent := 0.0
		if page.Percent != nil {
			percent = *page.Percent
		} else if limit := parseAmount(page.Limit); limit > 0 {
			percent = parseAmount(page.Used) / limit * 100
		}
		if percent < 0 {
			percent = 0
		}
		status := normalizeStatus("", percent)
		if status == "exceeded" {
			snap.Status = "exceeded"
		} else if status == "warning" {
			snap.Status = "warning"
		}
		snap.Windows = append(snap.Windows, Window{
			Key:     "plan",
			Label:   "套餐用量",
			Percent: percent,
			Status:  status,
			Detail:  strings.TrimSpace(page.Used + " / " + page.Limit),
			ResetAt: page.ExpiresAt,
		})
	}

	if page.TokenTotal != "" {
		snap.Stats = append(snap.Stats, Stat{Label: "Token 总消耗", Value: page.TokenTotal})
	}
	if page.RequestCount != "" {
		snap.Stats = append(snap.Stats, Stat{Label: "请求次数", Value: page.RequestCount, Muted: true})
	}
	if len(snap.Windows) == 0 && len(snap.Stats) == 0 {
		return nil, fmt.Errorf("MiMo 未读取到套餐数据，请确认已登录并开通 Token Plan")
	}
	return snap, nil
}

// parseAmount 解析 "508,596,984" 这类带千分位的数字。
func parseAmount(s string) float64 {
	v, _ := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", ""), 64)
	return v
}

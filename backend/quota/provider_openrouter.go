package quota

import (
	"fmt"
	"strings"
)

func init() { Register(openrouterProvider{}) }

type openrouterProvider struct{}

func (openrouterProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "openrouter", Name: "OpenRouter", Description: "查询当前 API Key 的消费和单 Key 限额；普通 Key 无法读取账户总余额", Mode: ModeAPI, KeyLabel: "API Key", KeyHint: "sk-or-...", DocsURL: "https://openrouter.ai/docs/api_reference/limits"}
}

type openrouterKeyResponse struct {
	Data struct {
		Label          string   `json:"label"`
		Limit          *float64 `json:"limit"`
		LimitRemaining *float64 `json:"limit_remaining"`
		Usage          float64  `json:"usage"`
		UsageDaily     float64  `json:"usage_daily"`
		UsageMonthly   float64  `json:"usage_monthly"`
	} `json:"data"`
}

func (openrouterProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp openrouterKeyResponse
	if err := httpGetJSON("https://openrouter.ai/api/v1/key", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	d := resp.Data
	snap := &Snapshot{Provider: "openrouter", AccountName: d.Label, Plan: "API Key 限额", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{{Label: "累计用量", Value: fmt.Sprintf("$%.2f", d.Usage)}, {Label: "今日用量", Value: fmt.Sprintf("$%.2f", d.UsageDaily)}, {Label: "本月用量", Value: fmt.Sprintf("$%.2f", d.UsageMonthly)}}, UpdatedAt: now()}
	if d.Limit != nil {
		remaining := *d.Limit - d.Usage
		if d.LimitRemaining != nil {
			remaining = *d.LimitRemaining
		}
		percent := 0.0
		if *d.Limit > 0 {
			percent = d.Usage / *d.Limit * 100
		}
		snap.Status = normalizeStatus("", percent)
		snap.Windows = append(snap.Windows, Window{Key: "key-limit", Label: "Key 消费限额", Percent: percent, Status: snap.Status, Detail: fmt.Sprintf("$%.2f / $%.2f", d.Usage, *d.Limit)})
		snap.Balances = append(snap.Balances, Stat{Label: "Key 剩余额度", Value: fmt.Sprintf("$%.2f", remaining)})
	} else {
		snap.Summary = "此 Key 未设置消费上限"
	}
	return snap, nil
}

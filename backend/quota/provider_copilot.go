package quota

import (
	"fmt"
	"strings"
)

func init() { Register(copilotProvider{}) }

type copilotProvider struct{}

func (copilotProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "copilot", Name: "GitHub Copilot", Description: "查询 GitHub Copilot 的 Premium、聊天与补全额度；需要可访问 Copilot 内部用量接口的 GitHub 登录令牌", Mode: ModeAPI, KeyLabel: "GitHub 登录令牌", KeyHint: "ghu_...", DocsURL: "https://github.com/deviffyy/OpenQuota"}
}

type copilotQuota struct {
	Plan      string `json:"copilot_plan"`
	ResetDate string `json:"quota_reset_date"`
	Snapshots map[string]struct {
		Entitlement      float64  `json:"entitlement"`
		Remaining        float64  `json:"remaining"`
		PercentRemaining *float64 `json:"percent_remaining"`
		Unlimited        bool     `json:"unlimited"`
	} `json:"quota_snapshots"`
}

func (copilotProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp copilotQuota
	if err := httpGetJSON("https://api.github.com/copilot_internal/user", map[string]string{"Authorization": "token " + strings.TrimSpace(qc.Account.Key), "Accept": "application/json", "X-GitHub-Api-Version": "2025-04-01", "User-Agent": "QuotaPanel/1.0"}, &resp); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "copilot", Plan: resp.Plan, Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	for _, item := range []struct{ key, label string }{{"premium_interactions", "Premium 请求"}, {"premium_requests", "Premium 请求"}, {"chat", "聊天"}, {"completions", "补全"}} {
		v, ok := resp.Snapshots[item.key]
		if !ok {
			continue
		}
		if item.key == "premium_requests" && len(snap.Windows) > 0 {
			continue
		}
		if v.Unlimited || v.Entitlement < 0 || v.Remaining < 0 {
			snap.Stats = append(snap.Stats, Stat{Label: item.label, Value: "不限量"})
			continue
		}
		if v.Entitlement == 0 {
			continue
		}
		used := v.Entitlement - v.Remaining
		percent := used / v.Entitlement * 100
		if v.PercentRemaining != nil {
			percent = 100 - *v.PercentRemaining
		}
		percent = max(0, min(100, percent))
		status := normalizeStatus("", percent)
		if status == "exceeded" {
			snap.Status = "exceeded"
		} else if status == "warning" && snap.Status == "ok" {
			snap.Status = "warning"
		}
		snap.Windows = append(snap.Windows, Window{Key: item.key, Label: item.label, Percent: percent, Status: status, Detail: fmt.Sprintf("%.0f / %.0f", used, v.Entitlement), ResetAt: formatISODisplay(resp.ResetDate)})
	}
	if len(snap.Windows) == 0 && len(snap.Stats) == 0 {
		return nil, fmt.Errorf("Copilot 未返回可识别的用量")
	}
	return snap, nil
}

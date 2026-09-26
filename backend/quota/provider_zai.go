package quota

import (
	"fmt"
	"strings"
)

func init() { Register(zaiProvider{}) }

type zaiProvider struct{}

func (zaiProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "zai", Name: "Z.ai Coding Plan", Description: "查询 GLM Coding Plan 的滚动及每周用量", Mode: ModeAPI, KeyLabel: "API Key", KeyHint: "请输入 Z.ai API Key", DocsURL: "https://github.com/deviffyy/OpenQuota"}
}

type zaiQuota struct {
	Data struct {
		Limits []struct {
			Type          string  `json:"type"`
			Name          string  `json:"name"`
			Unit          float64 `json:"unit"`
			Number        float64 `json:"number"`
			Percentage    float64 `json:"percentage"`
			NextResetTime int64   `json:"nextResetTime"`
			CurrentValue  float64 `json:"currentValue"`
			Usage         float64 `json:"usage"`
		} `json:"limits"`
	} `json:"data"`
}

func (zaiProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp zaiQuota
	if err := httpGetJSON("https://api.z.ai/api/monitor/usage/quota/limit", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "zai", Plan: "GLM Coding Plan", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	for _, v := range resp.Data.Limits {
		kind := v.Type
		if kind == "" {
			kind = v.Name
		}
		if kind != "TOKENS_LIMIT" && kind != "TIME_LIMIT" {
			continue
		}
		key, label := "session", "滚动窗口"
		percent := v.Percentage
		detail := ""
		if kind == "TIME_LIMIT" {
			key, label = "search", "联网搜索"
			if v.Usage > 0 {
				percent = v.CurrentValue / v.Usage * 100
			}
			detail = fmt.Sprintf("%.0f / %.0f", v.CurrentValue, v.Usage)
		} else {
			var ok bool
			key, label, ok = zaiWindow(v.Unit, v.Number)
			if !ok {
				continue
			}
		}
		percent = max(0, min(100, percent))
		status := normalizeStatus("", percent)
		if status == "exceeded" {
			snap.Status = "exceeded"
		} else if status == "warning" && snap.Status == "ok" {
			snap.Status = "warning"
		}
		snap.Windows = append(snap.Windows, Window{Key: key, Label: label, Percent: percent, Status: status, Detail: detail, ResetAt: formatEpochMillis(v.NextResetTime)})
	}
	if len(snap.Windows) == 0 {
		return nil, fmt.Errorf("Z.ai 未返回 Coding Plan 额度")
	}
	return snap, nil
}

func zaiWindow(unit, count float64) (string, string, bool) {
	if count <= 0 {
		return "", "", false
	}
	days := count
	switch unit {
	case 3:
		days /= 24
	case 4:
	case 5:
		days *= 30
	case 6:
		days *= 7
	default:
		return "", "", false
	}
	if days >= 1 {
		return "weekly", "长期窗口", true
	}
	return "session", "滚动窗口", true
}

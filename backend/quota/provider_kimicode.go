package quota

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func init() { Register(kimiCodeProvider{}) }

type kimiCodeProvider struct{}

func (kimiCodeProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "kimi-code", Name: "Kimi Coding Plan", Description: "查询 Kimi Coding Plan 订阅用量，与开放平台 API 余额分开", Mode: ModeAPI, KeyLabel: "Coding Plan Key", KeyHint: "请输入 Kimi Coding Plan Key", DocsURL: "https://github.com/deviffyy/OpenQuota"}
}

type kimiCodeUsage struct {
	Usage struct {
		Limit     string `json:"limit"`
		Used      string `json:"used"`
		ResetTime string `json:"resetTime"`
	} `json:"usage"`
	Limits []struct {
		Window struct {
			Duration int    `json:"duration"`
			TimeUnit string `json:"timeUnit"`
		} `json:"window"`
		Detail struct {
			Limit     string `json:"limit"`
			Remaining string `json:"remaining"`
			ResetTime string `json:"resetTime"`
		} `json:"detail"`
	} `json:"limits"`
}

func (kimiCodeProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp kimiCodeUsage
	if err := httpGetJSON("https://api.kimi.com/coding/v1/usages", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "kimi-code", Plan: "Kimi Coding Plan", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	add := func(key, label string, used, limit float64, reset string) {
		if limit <= 0 || math.IsNaN(limit) || math.IsNaN(used) || used < 0 {
			return
		}
		percent := max(0, min(100, used/limit*100))
		status := normalizeStatus("", percent)
		if status == "exceeded" {
			snap.Status = "exceeded"
		} else if status == "warning" && snap.Status == "ok" {
			snap.Status = "warning"
		}
		snap.Windows = append(snap.Windows, Window{Key: key, Label: label, Percent: percent, Status: status, Detail: fmt.Sprintf("%.0f / %.0f", used, limit), ResetAt: formatISODisplay(reset)})
	}
	used, _ := strconv.ParseFloat(resp.Usage.Used, 64)
	limit, _ := strconv.ParseFloat(resp.Usage.Limit, 64)
	add("weekly", "每周", used, limit, resp.Usage.ResetTime)
	for i, w := range resp.Limits {
		limit, _ := strconv.ParseFloat(w.Detail.Limit, 64)
		remaining, _ := strconv.ParseFloat(w.Detail.Remaining, 64)
		label := kimiWindowLabel(w.Window.Duration, w.Window.TimeUnit)
		add(fmt.Sprintf("window-%d", i), label, max(0, limit-remaining), limit, w.Detail.ResetTime)
	}
	if len(snap.Windows) == 0 {
		return nil, fmt.Errorf("Kimi 未返回 Coding Plan 用量")
	}
	return snap, nil
}

func kimiWindowLabel(duration int, unit string) string {
	minutes := duration
	switch unit {
	case "TIME_UNIT_SECOND":
		minutes /= 60
	case "TIME_UNIT_HOUR":
		minutes *= 60
	case "TIME_UNIT_MINUTE":
	default:
		return "滚动窗口"
	}
	if minutes > 0 && minutes%60 == 0 {
		return fmt.Sprintf("%d 小时", minutes/60)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d 分钟", minutes)
	}
	return "滚动窗口"
}

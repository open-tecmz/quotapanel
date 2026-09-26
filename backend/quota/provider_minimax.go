package quota

import (
	"fmt"
	"strings"
)

func init() { Register(minimaxProvider{}) }

type minimaxProvider struct{}

func (minimaxProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "minimax", Name: "MiniMax Token Plan", Description: "查询 MiniMax Token Plan 的滚动及每周剩余额度", Mode: ModeAPI, KeyLabel: "API Key", KeyHint: "请输入 MiniMax API Key", DocsURL: "https://github.com/deviffyy/OpenQuota"}
}

type minimaxRemains struct {
	BaseResp struct {
		StatusCode int    `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
	Models []struct {
		Name              string  `json:"model_name"`
		WeeklyBoost       float64 `json:"weekly_boost_permille"`
		IntervalRemaining float64 `json:"current_interval_remaining_percent"`
		WeeklyRemaining   float64 `json:"current_weekly_remaining_percent"`
		IntervalStatus    int     `json:"current_interval_status"`
		WeeklyStatus      int     `json:"current_weekly_status"`
		EndTime           int64   `json:"end_time"`
		WeeklyEndTime     int64   `json:"weekly_end_time"`
	} `json:"model_remains"`
}

func (minimaxProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp minimaxRemains
	if err := httpGetJSON("https://www.minimax.io/v1/token_plan/remains", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	if resp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("MiniMax: %s", resp.BaseResp.StatusMsg)
	}
	snap := &Snapshot{Provider: "minimax", Plan: "Token Plan", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	for _, model := range resp.Models {
		if model.Name != "general" {
			continue
		}
		for _, window := range []struct {
			key, label string
			remaining  float64
			allowance  float64
			state      int
			end        int64
		}{{"session", "滚动窗口", model.IntervalRemaining, 100, model.IntervalStatus, model.EndTime}, {"weekly", "每周", model.WeeklyRemaining, weeklyAllowance(model.WeeklyBoost), model.WeeklyStatus, model.WeeklyEndTime}} {
			percent := (window.allowance - window.remaining) / window.allowance * 100
			if window.state == 3 {
				percent = 0
			} else if window.remaining < 0 || window.remaining > window.allowance {
				return nil, fmt.Errorf("MiniMax 返回的剩余额度无效")
			}
			status := normalizeStatus("", percent)
			if status == "exceeded" {
				snap.Status = "exceeded"
			} else if status == "warning" && snap.Status == "ok" {
				snap.Status = "warning"
			}
			label := window.label
			if window.state == 3 {
				label += "（不限量）"
			}
			snap.Windows = append(snap.Windows, Window{Key: window.key, Label: label, Percent: percent, Status: status, ResetAt: formatEpochMillis(window.end)})
		}
	}
	if len(snap.Windows) == 0 {
		return nil, fmt.Errorf("MiniMax 未返回 Token Plan 用量")
	}
	return snap, nil
}

func weeklyAllowance(boost float64) float64 {
	if boost <= 0 {
		return 100
	}
	return boost / 10
}

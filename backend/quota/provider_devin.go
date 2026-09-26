package quota

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func init() { Register(devinProvider{}) }

type devinProvider struct{}

func (devinProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "devin", Name: "Devin / Windsurf", Description: "查询 Devin 扩展账号的每日、每周额度及超额余额", Mode: ModeAPI, KeyLabel: "Devin API Key", KeyHint: "请输入 Devin 扩展 API Key", DocsURL: "https://github.com/deviffyy/OpenQuota"}
}

type devinStatus struct {
	UserStatus struct {
		PlanStatus struct {
			PlanInfo struct {
				PlanName       string `json:"planName"`
				HideDailyQuota bool   `json:"hideDailyQuota"`
			} `json:"planInfo"`
			DailyRemaining  json.RawMessage `json:"dailyQuotaRemainingPercent"`
			WeeklyRemaining json.RawMessage `json:"weeklyQuotaRemainingPercent"`
			DailyReset      json.RawMessage `json:"dailyQuotaResetAtUnix"`
			WeeklyReset     json.RawMessage `json:"weeklyQuotaResetAtUnix"`
			OverageBalance  json.RawMessage `json:"overageBalanceMicros"`
		} `json:"planStatus"`
	} `json:"userStatus"`
}

func (devinProvider) Query(qc *QueryContext) (*Snapshot, error) {
	key := strings.TrimSpace(qc.Account.Key)
	body, err := json.Marshal(map[string]any{"metadata": map[string]string{"apiKey": key, "ideName": "devin", "ideVersion": "1.108.2", "extensionName": "devin", "extensionVersion": "1.108.2", "locale": "en"}})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, "https://server.codeium.com/exa.seat_management_pb.SeatManagementService/GetUserStatus", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Devin 额度查询失败：HTTP %d", resp.StatusCode)
	}
	var result devinStatus
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 Devin 用量失败: %w", err)
	}
	return mapDevinStatus(result)
}

func mapDevinStatus(result devinStatus) (*Snapshot, error) {
	plan := result.UserStatus.PlanStatus
	snap := &Snapshot{Provider: "devin", Plan: plan.PlanInfo.PlanName, Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	add := func(key, label string, remaining, reset json.RawMessage) {
		value, ok := devinNumber(remaining)
		if !ok {
			return
		}
		percent := max(0, min(100, 100-value))
		status := normalizeStatus("", percent)
		if status == "exceeded" || status == "warning" && snap.Status == "ok" {
			snap.Status = status
		}
		resetAt := ""
		if seconds, ok := devinNumber(reset); ok && seconds > 0 {
			resetAt = time.Unix(int64(seconds), 0).Local().Format("01-02 15:04")
		}
		snap.Windows = append(snap.Windows, Window{Key: key, Label: label, Percent: percent, Status: status, ResetAt: resetAt})
	}
	if !plan.PlanInfo.HideDailyQuota {
		add("daily", "每日", plan.DailyRemaining, plan.DailyReset)
	}
	if len(plan.WeeklyRemaining) > 0 {
		add("weekly", "每周", plan.WeeklyRemaining, plan.WeeklyReset)
	} else if plan.PlanInfo.HideDailyQuota {
		add("weekly", "每周", plan.DailyRemaining, plan.WeeklyReset)
	}
	if micros, ok := devinNumber(plan.OverageBalance); ok {
		snap.Balances = append(snap.Balances, Stat{Label: "超额余额", Value: fmt.Sprintf("$%.2f", max(0, micros)/1_000_000)})
	}
	if len(snap.Windows) == 0 && len(snap.Balances) == 0 {
		return nil, fmt.Errorf("Devin 未返回可识别的额度")
	}
	return snap, nil
}

func devinNumber(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var number float64
	if json.Unmarshal(raw, &number) == nil {
		return number, !math.IsNaN(number) && !math.IsInf(number, 0)
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		return value, err == nil && !math.IsNaN(value) && !math.IsInf(value, 0)
	}
	return 0, false
}

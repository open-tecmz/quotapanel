package quota

import (
	"fmt"
	"strconv"
	"strings"
)

// deepseekAPIBase 为 DeepSeek 开放平台 API 基址。
const deepseekAPIBase = "https://api.deepseek.com"

func init() { Register(deepseekProvider{}) }

type deepseekProvider struct{}

func (deepseekProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "deepseek", Name: "DeepSeek API", Description: "查询 DeepSeek 开放平台余额（不包含网页聊天套餐）", Mode: ModeAPI, KeyLabel: "API Key", KeyHint: "sk-...", DocsURL: "https://api-docs.deepseek.com/api/get-user-balance"}
}

type deepseekBalance struct {
	IsAvailable  bool `json:"is_available"`
	BalanceInfos []struct {
		Currency string `json:"currency"`
		Total    string `json:"total_balance"`
		Granted  string `json:"granted_balance"`
		ToppedUp string `json:"topped_up_balance"`
	} `json:"balance_infos"`
}

func (deepseekProvider) Query(qc *QueryContext) (*Snapshot, error) {
	return queryDeepseek(qc, deepseekAPIBase)
}

func queryDeepseek(qc *QueryContext, apiBase string) (*Snapshot, error) {
	var resp deepseekBalance
	if err := httpGetJSON(apiBase+"/user/balance", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	if len(resp.BalanceInfos) == 0 {
		return nil, fmt.Errorf("DeepSeek 未返回余额数据")
	}
	snap := &Snapshot{Provider: "deepseek", Plan: "API 余额", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	if !resp.IsAvailable {
		snap.Status = "exceeded"
	}
	for _, b := range resp.BalanceInfos {
		currency := strings.TrimSpace(b.Currency)
		snap.Balances = append(snap.Balances, Stat{Label: "可用余额", Value: money(b.Total, currency)})
		snap.Stats = append(snap.Stats,
			Stat{Label: "赠送余额", Value: money(b.Granted, currency), Muted: true},
			Stat{Label: "充值余额", Value: money(b.ToppedUp, currency), Muted: true},
		)
		if n, err := strconv.ParseFloat(b.Total, 64); err == nil && n <= 0 {
			snap.Status = "exceeded"
		}
	}
	return snap, nil
}

// money 按币种为金额加上符号，空值统一显示为 0。
func money(amount, currency string) string {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		amount = "0"
	}
	symbol := ""
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "CNY":
		symbol = "¥"
	case "USD":
		symbol = "$"
	}
	if symbol == "" {
		return amount + " " + strings.TrimSpace(currency)
	}
	return symbol + amount
}

package quota

import (
	"fmt"
	"strings"
)

func init() { Register(kimiProvider{}) }

type kimiProvider struct{}

func (kimiProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "kimi", Name: "Kimi API", Description: "查询 Kimi 国内开放平台余额（不包含 Kimi 网页会员）", Mode: ModeAPI, KeyLabel: "API Key", KeyHint: "sk-...", DocsURL: "https://platform.kimi.com/docs/api/balance"}
}

type kimiBalance struct {
	Status bool `json:"status"`
	Data   struct {
		Available float64 `json:"available_balance"`
		Voucher   float64 `json:"voucher_balance"`
		Cash      float64 `json:"cash_balance"`
	} `json:"data"`
}

func (kimiProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp kimiBalance
	if err := httpGetJSON("https://api.moonshot.cn/v1/users/me/balance", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	if !resp.Status {
		return nil, fmt.Errorf("Kimi 返回余额查询失败")
	}
	status := "ok"
	if resp.Data.Available <= 0 {
		status = "exceeded"
	}
	return &Snapshot{Provider: "kimi", Plan: "API 余额", Status: status, Windows: []Window{}, Balances: []Stat{{Label: "可用余额", Value: fmt.Sprintf("¥%.2f", resp.Data.Available)}}, Stats: []Stat{{Label: "代金券", Value: fmt.Sprintf("¥%.2f", resp.Data.Voucher)}, {Label: "现金", Value: fmt.Sprintf("¥%.2f", resp.Data.Cash)}}, UpdatedAt: now()}, nil
}

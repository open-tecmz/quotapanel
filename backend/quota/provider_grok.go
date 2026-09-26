package quota

import (
	"fmt"
	"strings"
)

func init() { Register(grokProvider{}) }

type grokProvider struct{}

func (grokProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "grok", Name: "Grok CLI", Description: "查询 Grok CLI 订阅周期额度；需要 Grok CLI OAuth access token", Mode: ModeAPI, KeyLabel: "OAuth Access Token", KeyHint: "请输入 Grok CLI access token", DocsURL: "https://github.com/deviffyy/OpenQuota"}
}

type grokCredits struct {
	Config struct {
		CreditUsagePercent float64 `json:"creditUsagePercent"`
		CurrentPeriod      struct {
			Type  string `json:"type"`
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"currentPeriod"`
	} `json:"config"`
}

func (grokProvider) Query(qc *QueryContext) (*Snapshot, error) {
	var resp grokCredits
	if err := httpGetJSON("https://cli-chat-proxy.grok.com/v1/billing?format=credits", map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key), "X-XAI-Token-Auth": "xai-grok-cli"}, &resp); err != nil {
		return nil, err
	}
	if resp.Config.CurrentPeriod.Type == "" {
		return nil, fmt.Errorf("Grok 未返回订阅周期")
	}
	status := normalizeStatus("", resp.Config.CreditUsagePercent)
	return &Snapshot{Provider: "grok", Plan: "Grok CLI", Status: status, Windows: []Window{{Key: "period", Label: "当前周期", Percent: resp.Config.CreditUsagePercent, Status: status, ResetAt: formatISODisplay(resp.Config.CurrentPeriod.End)}}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}, nil
}

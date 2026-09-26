package quota

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func init() { Register(openaiProvider{}) }

type openaiProvider struct{}

func (openaiProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "openai-api", Name: "OpenAI API", Description: "使用组织 Admin Key 查询近 30 天 API 费用；不包含 ChatGPT Plus/Pro 或 Codex 订阅额度", Mode: ModeAPI, KeyLabel: "Admin API Key", KeyHint: "sk-admin-...", DocsURL: "https://developers.openai.com/api/reference/resources/admin/subresources/organization/subresources/usage/methods/costs"}
}

type openaiCostsResponse struct {
	Data []struct {
		Results []struct {
			Amount struct {
				Value    float64 `json:"value"`
				Currency string  `json:"currency"`
			} `json:"amount"`
		} `json:"results"`
	} `json:"data"`
	HasMore bool `json:"has_more"`
}

func (openaiProvider) Query(qc *QueryContext) (*Snapshot, error) {
	start := time.Now().UTC().AddDate(0, 0, -30).Unix()
	u := "https://api.openai.com/v1/organization/costs?start_time=" + url.QueryEscape(fmt.Sprint(start)) + "&limit=30"
	var resp openaiCostsResponse
	if err := httpGetJSON(u, map[string]string{"Authorization": "Bearer " + strings.TrimSpace(qc.Account.Key)}, &resp); err != nil {
		return nil, err
	}
	var total float64
	for _, day := range resp.Data {
		for _, result := range day.Results {
			if result.Amount.Currency == "usd" {
				total += result.Amount.Value
			}
		}
	}
	summary := "近 30 天组织 API 费用；不代表订阅剩余额度"
	if resp.HasMore {
		summary = "费用结果尚有更多页面，当前显示部分数据"
	}
	return &Snapshot{Provider: "openai-api", Plan: "组织 API 费用", Status: "active", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{{Label: "近 30 天费用", Value: fmt.Sprintf("$%.2f", total)}}, Summary: summary, UpdatedAt: now()}, nil
}

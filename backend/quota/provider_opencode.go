package quota

import (
	"fmt"
	"strings"
)

// opencodeUsageURL 是 OpenCode Go 套餐的用量接口。
const opencodeUsageURL = "https://opencode.ai/zen/go/v1/usage"

func init() {
	Register(opencodeProvider{})
}

type opencodeProvider struct{}

func (opencodeProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          "opencode",
		Name:        "OpenCode Go",
		Description: "OpenCode 官方 Go 套餐，查看滚动（5小时）/每周/月度用量",
		Mode:        ModeAPI,
		KeyLabel:    "API Key",
		KeyHint:     "sk-...",
		DocsURL:     "https://opencode.ai",
	}
}

type opencodeWindow struct {
	Status   string  `json:"status"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resetsAt"`
}

type opencodeUsageResponse struct {
	Usage struct {
		Rolling opencodeWindow `json:"rolling"`
		Weekly  opencodeWindow `json:"weekly"`
		Monthly opencodeWindow `json:"monthly"`
	} `json:"usage"`
}

func (opencodeProvider) Query(qc *QueryContext) (*Snapshot, error) {
	key := strings.TrimSpace(qc.Account.Key)
	if key == "" {
		return nil, fmt.Errorf("API Key 不能为空")
	}
	var resp opencodeUsageResponse
	err := httpGetJSON(opencodeUsageURL, map[string]string{
		"Authorization": "Bearer " + key,
		"User-Agent":    "QuotaPanel-Quota/1.0",
	}, &resp)
	if err != nil {
		return nil, err
	}

	u := resp.Usage
	windows := []Window{
		{
			Key:     "rolling",
			Label:   "滚动（5小时）",
			Percent: u.Rolling.Percent,
			Status:  normalizeStatus(u.Rolling.Status, u.Rolling.Percent),
			ResetAt: formatISODisplay(u.Rolling.ResetsAt),
		},
		{
			Key:     "weekly",
			Label:   "每周",
			Percent: u.Weekly.Percent,
			Status:  normalizeStatus(u.Weekly.Status, u.Weekly.Percent),
			ResetAt: formatISODisplay(u.Weekly.ResetsAt),
		},
		{
			Key:     "monthly",
			Label:   "月度",
			Percent: u.Monthly.Percent,
			Status:  normalizeStatus(u.Monthly.Status, u.Monthly.Percent),
			ResetAt: formatISODisplay(u.Monthly.ResetsAt),
		},
	}

	snap := &Snapshot{
		Provider:  "opencode",
		Plan:      "OpenCode Go",
		Status:    normalizeStatus(u.Monthly.Status, u.Monthly.Percent),
		Windows:   windows,
		Balances:  []Stat{},
		Stats:     []Stat{},
		Summary:   opencodeSummary(u.Monthly.Percent),
		UpdatedAt: now(),
	}
	return snap, nil
}

func opencodeSummary(monthlyPercent float64) string {
	switch {
	case monthlyPercent >= 100:
		return "月度额度已用尽，等待重置后可继续使用"
	case monthlyPercent >= 80:
		return fmt.Sprintf("月度额度已用 %.0f%%，余量偏低", monthlyPercent)
	default:
		return fmt.Sprintf("额度充足，月度剩余 %.0f%%", 100-monthlyPercent)
	}
}

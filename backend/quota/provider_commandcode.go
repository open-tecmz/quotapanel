package quota

import (
	"fmt"
	"net/url"
	"strings"
)

// commandcode API 基址与接口路径。
const (
	commandcodeAPIBase     = "https://api.commandcode.ai"
	commandcodePathWhoami  = "/alpha/whoami"
	commandcodePathCredits = "/alpha/billing/credits"
	commandcodePathSubs    = "/alpha/billing/subscriptions"
	commandcodePathSummary = "/alpha/usage/summary"
)

func init() {
	Register(commandcodeProvider{})
}

type commandcodeProvider struct{}

func (commandcodeProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          "commandcode",
		Name:        "Command Goat",
		Description: "Command Code（CommandGoat）订阅套餐，查看账号、套餐与窗口用量",
		Mode:        ModeAPI,
		KeyLabel:    "API Key",
		KeyHint:     "user_...",
		DocsURL:     "https://commandcode.ai/docs/resources/usage-limits",
	}
}

// commandcodePlan 描述一个套餐的展示名与月度额度。
type commandcodePlan struct {
	Name    string
	Monthly float64
}

var commandcodePlans = map[string]commandcodePlan{
	"individual-go":       {"Go", 10},
	"individual-goat":     {"GOAT", 70},
	"individual-pro-v1":   {"Pro", 80},
	"individual-pro":      {"Pro", 80},
	"individual-provider": {"Provider", 0},
	"individual-max-10x":  {"Max 10x", 150},
	"individual-max-20x":  {"Max 20x", 300},
	"individual-max":      {"Max", 150},
	"individual-ultra":    {"Ultra", 300},
	"teams-pro":           {"Teams Pro", 40},
	"teams-enterprise":    {"Enterprise", 0},
}

// 只认已知套餐 ID，避免新版本套餐被旧套餐前缀误判。
func lookupCommandcodePlan(planID string) (commandcodePlan, bool) {
	if planID == "" {
		return commandcodePlan{}, false
	}
	normalized := strings.ToLower(strings.ReplaceAll(planID, "_", "-"))
	plan, found := commandcodePlans[normalized]
	if !found {
		return commandcodePlan{Name: planID}, true
	}
	return plan, true
}

type commandcodeWhoami struct {
	User *struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		UserName string `json:"userName"`
	} `json:"user"`
	Org *struct {
		ID    string `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	} `json:"org"`
}

type commandcodeWindow struct {
	Used     float64 `json:"used"`
	Cap      float64 `json:"cap"`
	Exceeded bool    `json:"exceeded"`
	ResetAt  int64   `json:"resetAt"`
}

type commandcodeCreditsResp struct {
	Credits struct {
		MonthlyCredits   float64 `json:"monthlyCredits"`
		PurchasedCredits float64 `json:"purchasedCredits"`
		FreeCredits      float64 `json:"freeCredits"`
		PlanID           string  `json:"planId"`
	} `json:"credits"`
	WindowLimits struct {
		FiveHour *commandcodeWindow `json:"fiveHour"`
		Weekly   *commandcodeWindow `json:"weekly"`
	} `json:"windowLimits"`
}

type commandcodeSubsResp struct {
	Data *struct {
		Status             string `json:"status"`
		PlanID             string `json:"planId"`
		CurrentPeriodStart string `json:"currentPeriodStart"`
		CurrentPeriodEnd   string `json:"currentPeriodEnd"`
	} `json:"data"`
}

type commandcodeSummaryResp struct {
	TotalCount          int      `json:"totalCount"`
	TotalCost           float64  `json:"totalCost"`
	SuccessRate         float64  `json:"successRate"`
	CompletedCount      int      `json:"completedCount"`
	FailedCount         int      `json:"failedCount"`
	TotalTokensIn       int64    `json:"totalTokensIn"`
	TotalTokensOut      int64    `json:"totalTokensOut"`
	TotalTokens         int64    `json:"totalTokens"`
	TotalMonthlyCredits *float64 `json:"totalMonthlyCredits"`
	PeriodBasis         string   `json:"periodBasis"`
}

func (commandcodeProvider) Query(qc *QueryContext) (*Snapshot, error) {
	return queryCommandcode(qc, commandcodeAPIBase)
}

func queryCommandcode(qc *QueryContext, apiBase string) (*Snapshot, error) {
	key := strings.TrimSpace(qc.Account.Key)
	if key == "" {
		return nil, fmt.Errorf("API Key 不能为空")
	}
	headers := map[string]string{
		"Authorization": "Bearer " + key,
		"x-api-key":     key,
		"User-Agent":    "QuotaPanel-Quota/1.0",
	}

	var whoami commandcodeWhoami
	if err := httpGetJSON(apiBase+commandcodePathWhoami+"?limits=1", headers, &whoami); err != nil {
		return nil, fmt.Errorf("获取账号信息失败: %w", err)
	}
	orgQuery := ""
	if whoami.Org != nil && whoami.Org.ID != "" {
		orgQuery = "?orgId=" + url.QueryEscape(whoami.Org.ID)
	}
	var credits commandcodeCreditsResp
	if err := httpGetJSON(apiBase+commandcodePathCredits+orgQuery, headers, &credits); err != nil {
		return nil, fmt.Errorf("获取额度信息失败: %w", err)
	}
	var subs commandcodeSubsResp
	if err := httpGetJSON(apiBase+commandcodePathSubs+orgQuery, headers, &subs); err != nil {
		return nil, fmt.Errorf("获取套餐信息失败: %w", err)
	}
	summaryQuery := url.Values{}
	if whoami.Org != nil && whoami.Org.ID != "" {
		summaryQuery.Set("orgId", whoami.Org.ID)
	}
	if subs.Data != nil && subs.Data.CurrentPeriodStart != "" {
		summaryQuery.Set("since", subs.Data.CurrentPeriodStart)
	}
	summaryURL := apiBase + commandcodePathSummary
	if len(summaryQuery) > 0 {
		summaryURL += "?" + summaryQuery.Encode()
	}
	var summary commandcodeSummaryResp
	if err := httpGetJSON(summaryURL, headers, &summary); err != nil {
		return nil, fmt.Errorf("获取用量汇总失败: %w", err)
	}

	snap := &Snapshot{
		Provider:  "commandcode",
		Plan:      "",
		Windows:   []Window{},
		Balances:  []Stat{},
		Stats:     []Stat{},
		UpdatedAt: now(),
	}

	if whoami.User != nil {
		snap.AccountName = whoami.User.UserName
		if snap.AccountName == "" {
			snap.AccountName = whoami.User.Name
		}
		if whoami.User.Email != "" {
			snap.AccountName = fmt.Sprintf("%s <%s>", snap.AccountName, whoami.User.Email)
		}
	}
	if whoami.Org != nil {
		orgName := whoami.Org.Login
		if orgName == "" {
			orgName = whoami.Org.Name
		}
		snap.AccountName = fmt.Sprintf("%s（组织）", orgName)
	}

	planID := credits.Credits.PlanID
	if subs.Data != nil {
		if subs.Data.PlanID != "" {
			planID = subs.Data.PlanID
		}
		snap.Status = subs.Data.Status
		if subs.Data.CurrentPeriodStart != "" && subs.Data.CurrentPeriodEnd != "" {
			snap.Summary = fmt.Sprintf(
				"计费周期 %s ~ %s",
				formatISODisplay(subs.Data.CurrentPeriodStart),
				formatISODisplay(subs.Data.CurrentPeriodEnd),
			)
		}
	} else if credits.Credits.MonthlyCredits > 0 {
		snap.Status = "active"
	}
	plan, hasPlan := lookupCommandcodePlan(planID)
	if hasPlan {
		snap.Plan = plan.Name
	}

	for _, w := range []struct {
		key   string
		label string
		win   *commandcodeWindow
	}{
		{"rolling", "滚动（5小时）", credits.WindowLimits.FiveHour},
		{"weekly", "每周", credits.WindowLimits.Weekly},
	} {
		if w.win == nil {
			continue
		}
		pct := 0.0
		if w.win.Cap > 0 {
			pct = w.win.Used / w.win.Cap * 100
		}
		status := normalizeStatus("", pct)
		if w.win.Exceeded {
			status = "exceeded"
		}
		snap.Windows = append(snap.Windows, Window{
			Key:     w.key,
			Label:   w.label,
			Percent: pct,
			Status:  status,
			Detail:  fmt.Sprintf("$%.2f / $%.2f", w.win.Used, w.win.Cap),
			ResetAt: formatEpochMillis(w.win.ResetAt),
		})
	}

	c := credits.Credits
	available := c.MonthlyCredits + c.PurchasedCredits + c.FreeCredits
	hasMonthlyAllowance := plan.Monthly > 0 || (c.MonthlyCredits > 0 && plan.Name != "Provider" && plan.Name != "Enterprise")
	if hasMonthlyAllowance && summary.TotalMonthlyCredits != nil && summary.PeriodBasis == "billing-period" {
		used := *summary.TotalMonthlyCredits
		monthlyTotal := used + c.MonthlyCredits
		if monthlyTotal > 0 {
			percent := used / monthlyTotal * 100
			resetAt := "-"
			if subs.Data != nil {
				resetAt = formatISODisplay(subs.Data.CurrentPeriodEnd)
			}
			snap.Windows = append(snap.Windows, Window{
				Key:     "monthly",
				Label:   "月度",
				Percent: percent,
				Status:  normalizeStatus("", percent),
				Detail:  fmt.Sprintf("$%.2f / $%.2f", used, monthlyTotal),
				ResetAt: resetAt,
			})
		}
		snap.Balances = append(snap.Balances, Stat{
			Label: "本期月度额度",
			Value: fmt.Sprintf("剩余 $%.2f / 合计 $%.2f（已用 $%.2f）", c.MonthlyCredits, monthlyTotal, used),
		})
	} else if hasMonthlyAllowance {
		snap.Balances = append(snap.Balances, Stat{Label: "月度额度", Value: fmt.Sprintf("$%.2f", c.MonthlyCredits)})
	}
	snap.Balances = append(snap.Balances,
		Stat{Label: "额外额度", Value: fmt.Sprintf("$%.2f", c.PurchasedCredits), Muted: true},
		Stat{Label: "赠送额度", Value: fmt.Sprintf("$%.2f", c.FreeCredits), Muted: true},
		Stat{Label: "可用合计", Value: fmt.Sprintf("$%.2f", available)},
	)

	snap.Stats = append(snap.Stats,
		Stat{Label: "请求数", Value: fmt.Sprintf("%d（成功 %d / 失败 %d）", summary.TotalCount, summary.CompletedCount, summary.FailedCount)},
		Stat{Label: "成功率", Value: fmt.Sprintf("%.0f%%", summary.SuccessRate)},
		Stat{Label: "消耗金额", Value: fmt.Sprintf("$%.4f", summary.TotalCost)},
		Stat{Label: "Token", Value: fmt.Sprintf("输入 %d / 输出 %d / 合计 %d", summary.TotalTokensIn, summary.TotalTokensOut, summary.TotalTokens)},
	)

	if snap.Summary != "" {
		snap.Summary = fmt.Sprintf("%s · 可用合计 $%.2f", snap.Summary, available)
	} else {
		snap.Summary = fmt.Sprintf("可用合计 $%.2f", available)
	}
	if available <= 0 && plan.Monthly > 0 {
		snap.Status = "exceeded"
	}
	return snap, nil
}

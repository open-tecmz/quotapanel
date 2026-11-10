package quota

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// OpenDesign Cloud 的额度读取沿用控制台自身的只读接口：
//
//	GET /api/v1/workspaces/{id}/billing/preflight    Design / Coding Plan 窗口
//	GET /api/v1/workspaces/{id}/wallet/balance       钱包余额
//
// 推理 API Key（sk-amr_…）只能调用模型，官方尚未开放额度查询（nexu-io/open-design#8674），
// 这里要求 Control Key（vela_ctrl_…）并显式给出工作区 ID。
const opendesignAPIBase = "https://amr-api.open-design.ai/api/v1"

func init() { Register(opendesignProvider{}) }

type opendesignProvider struct{}

func (opendesignProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          "opendesign",
		Name:        "OpenDesign",
		Description: "查询 OpenDesign Cloud 的 Design / Coding Plan 5 小时与每周窗口、钱包余额；需要 Control Key（工作区 ID 可省略），推理 API Key 无法查询额度",
		Mode:        ModeAPI,
		KeyLabel:    "Control Key",
		KeyHint:     "vela_ctrl_...（需要指定工作区时填 workspaceId|vela_ctrl_...）",
		DocsURL:     "https://open-design.ai/help/billing/",
	}
}

// opendesignAmount 兼容额度接口里以字符串给出的金额与 credits（个别字段可能是数字）。
type opendesignAmount string

func (a *opendesignAmount) UnmarshalJSON(b []byte) error {
	raw := strings.TrimSpace(string(b))
	if raw == "null" {
		*a = ""
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*a = opendesignAmount(strings.TrimSpace(v))
		return nil
	}
	*a = opendesignAmount(raw)
	return nil
}

// opendesignWindow 是一个额度窗口；durationSeconds 用来区分 5 小时（18000）与每周（604800）。
type opendesignWindow struct {
	PolicyID         string           `json:"policyId"`
	DurationSeconds  int64            `json:"durationSeconds"`
	UsedCredits      opendesignAmount `json:"usedCredits"`
	LimitCredits     opendesignAmount `json:"limitCredits"`
	RemainingCredits opendesignAmount `json:"remainingCredits"`
	ResetsAt         *string          `json:"resetsAt"`
}

type opendesignPlan struct {
	WorkspaceID string             `json:"workspaceId"`
	Eligible    bool               `json:"eligible"`
	Tier        *string            `json:"tier"`
	Windows     []opendesignWindow `json:"windows"`
}

type opendesignPreflight struct {
	WorkspaceID       string           `json:"workspaceId"`
	WorkspaceMemberID string           `json:"workspaceMemberId"`
	BalanceUSD        opendesignAmount `json:"balanceUsd"`
	Funding           string           `json:"funding"`
	CodingPlan        *opendesignPlan  `json:"codingPlan"`
}

type opendesignWallet struct {
	BillingScopeVersion int              `json:"billingScopeVersion"`
	WorkspaceID         string           `json:"workspaceId"`
	WorkspaceMemberID   string           `json:"workspaceMemberId"`
	BalanceUSD          opendesignAmount `json:"balanceUsd"`
}

// opendesignBillingSummary 提供钱包余额、credits 与美元的换算比例和会员档位。
// 这些字段用于把窗口的 credits 折算成美元展示；读不到时退回 credits 文案。
type opendesignBillingSummary struct {
	BalanceUSD     opendesignAmount `json:"balanceUsd"`
	CreditsPerUSD  opendesignAmount `json:"creditsPerUsd"`
	MembershipTier *string          `json:"membershipTier"`
}

// opendesignCredential 是「工作区 ID + Control Key」的组合凭据。
// 账号模型只有 Key 一个字段，因此用 | 拼接；只填 Control Key 时尝试自动读取工作区。
type opendesignCredential struct {
	workspaceID string
	token       string
}

func parseOpenDesignCredential(raw string) (opendesignCredential, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return opendesignCredential{}, fmt.Errorf("Control Key 不能为空")
	}
	workspace, token := "", raw
	if i := strings.Index(raw, "|"); i >= 0 {
		workspace = strings.TrimSpace(raw[:i])
		token = strings.TrimSpace(raw[i+1:])
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[len("bearer "):])
	}
	if token == "" {
		return opendesignCredential{}, fmt.Errorf("Control Key 不能为空")
	}
	if strings.HasPrefix(strings.ToLower(token), "sk-amr") {
		return opendesignCredential{}, fmt.Errorf("OpenDesign 推理 API Key 无法查询额度，请改用 Control Key（vela_ctrl_…）")
	}
	if workspace != "" && !validOpenDesignID(workspace) {
		return opendesignCredential{}, fmt.Errorf("工作区 ID 格式不正确")
	}
	return opendesignCredential{workspaceID: workspace, token: token}, nil
}

// validOpenDesignID 校验工作区 ID，避免拼接出意外路径。
func validOpenDesignID(v string) bool {
	if v == "" || len(v) > 128 {
		return false
	}
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

func opendesignHeaders(token, workspaceID string) map[string]string {
	h := map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
		"User-Agent":    "QuotaPanel-Quota/1.0",
	}
	if workspaceID != "" {
		h["x-vela-workspace-id"] = workspaceID
	}
	return h
}

// opendesignDiscoverWorkspace 在只填了 Control Key 时读取账号当前工作区。
// 优先用 /workspaces/current（Control Key 可读），失败再退回工作区列表；
// 都读不到时要求用户按「工作区ID|Control Key」填写。
func opendesignDiscoverWorkspace(base, token string) (string, error) {
	var current struct {
		ActiveWorkspaceID string `json:"activeWorkspaceId"`
		WorkspaceID       string `json:"workspaceId"`
	}
	if err := httpGetJSON(base+"/workspaces/current", opendesignHeaders(token, ""), &current); err == nil {
		if validOpenDesignID(current.ActiveWorkspaceID) {
			return current.ActiveWorkspaceID, nil
		}
		if validOpenDesignID(current.WorkspaceID) {
			return current.WorkspaceID, nil
		}
	}
	var list struct {
		Items []struct {
			WorkspaceID string `json:"workspaceId"`
		} `json:"items"`
	}
	if err := httpGetJSON(base+"/workspaces", opendesignHeaders(token, ""), &list); err != nil {
		return "", fmt.Errorf("未能自动读取工作区：%w；请按「工作区ID|Control Key」填写", opendesignError(err))
	}
	for _, item := range list.Items {
		if validOpenDesignID(item.WorkspaceID) {
			return item.WorkspaceID, nil
		}
	}
	return "", fmt.Errorf("未找到可用的工作区，请按「工作区ID|Control Key」填写")
}

func (opendesignProvider) Query(qc *QueryContext) (*Snapshot, error) {
	return queryOpenDesign(qc, opendesignAPIBase)
}

// queryOpenDesign 把额度接口映射为统一快照；base 便于测试替换接口地址。
func queryOpenDesign(qc *QueryContext, base string) (*Snapshot, error) {
	if qc == nil || qc.Account == nil {
		return nil, fmt.Errorf("缺少账号信息")
	}
	cred, err := parseOpenDesignCredential(qc.Account.Key)
	if err != nil {
		return nil, err
	}
	if cred.workspaceID == "" {
		if cred.workspaceID, err = opendesignDiscoverWorkspace(base, cred.token); err != nil {
			return nil, err
		}
	}
	headers := opendesignHeaders(cred.token, cred.workspaceID)

	var preflight opendesignPreflight
	if err := httpGetJSON(fmt.Sprintf("%s/workspaces/%s/billing/preflight", base, cred.workspaceID), headers, &preflight); err != nil {
		return nil, opendesignError(err)
	}
	// 响应必须属于请求的工作区，避免把别处的额度显示到当前账号上。
	if preflight.WorkspaceID != "" && preflight.WorkspaceID != cred.workspaceID {
		return nil, fmt.Errorf("额度响应所属工作区与填写的工作区 ID 不一致")
	}
	if plan := preflight.CodingPlan; plan != nil && plan.WorkspaceID != "" && plan.WorkspaceID != cred.workspaceID {
		return nil, fmt.Errorf("额度响应所属工作区与填写的工作区 ID 不一致")
	}

	// 钱包余额与换算比例：任一接口失败都退回上一层的值，不把余额当成 0。
	balance := string(preflight.BalanceUSD)
	var summary opendesignBillingSummary
	summaryOK := httpGetJSON(fmt.Sprintf("%s/billing/summary", base), headers, &summary) == nil
	if summaryOK && summary.BalanceUSD != "" {
		balance = string(summary.BalanceUSD)
	}
	var wallet opendesignWallet
	if err := httpGetJSON(fmt.Sprintf("%s/workspaces/%s/wallet/balance", base, cred.workspaceID), headers, &wallet); err == nil {
		if (wallet.WorkspaceID == "" || wallet.WorkspaceID == cred.workspaceID) && wallet.BalanceUSD != "" {
			balance = string(wallet.BalanceUSD)
		}
	}
	creditsPerUSD := 0.0
	if summaryOK {
		if rate, ok := opendesignCredits(summary.CreditsPerUSD); ok && rate > 0 {
			creditsPerUSD = rate
		}
	}

	plan := preflight.CodingPlan
	tier := ""
	if plan != nil && plan.Tier != nil {
		tier = *plan.Tier
	}
	if tier == "" && summaryOK && summary.MembershipTier != nil {
		tier = *summary.MembershipTier
	}
	snap := &Snapshot{
		Provider:  "opendesign",
		Plan:      opendesignTierLabel(tier),
		Status:    "ok",
		Windows:   []Window{},
		Balances:  []Stat{},
		Stats:     []Stat{},
		UpdatedAt: now(),
	}

	if plan != nil {
		// 5 小时窗口只有 Go 档存在，顺序固定为短窗口在前。
		sort.SliceStable(plan.Windows, func(i, j int) bool {
			return plan.Windows[i].DurationSeconds < plan.Windows[j].DurationSeconds
		})
		for _, w := range plan.Windows {
			used, okUsed := opendesignCredits(w.UsedCredits)
			limit, okLimit := opendesignCredits(w.LimitCredits)
			if !okUsed || !okLimit || limit <= 0 {
				continue
			}
			percent := math.Round(min(100, max(0, used/limit*100))*10) / 10
			key, label := opendesignWindowMeta(w.DurationSeconds)
			reset := ""
			if w.ResetsAt != nil {
				reset = formatISODisplay(*w.ResetsAt)
			}
			window := Window{
				Key:     key,
				Label:   label,
				Percent: percent,
				Status:  normalizeStatus("", percent),
				Detail:  opendesignWindowDetail(used, limit, creditsPerUSD, w.UsedCredits, w.LimitCredits),
				ResetAt: reset,
			}
			if window.Status == "exceeded" {
				snap.Status = "exceeded"
			} else if window.Status == "warning" && snap.Status == "ok" {
				snap.Status = "warning"
			}
			snap.Windows = append(snap.Windows, window)
		}
	}

	if balance != "" {
		snap.Balances = append(snap.Balances, Stat{Label: "钱包余额", Value: opendesignUSD(balance)})
	}
	if preflight.Funding != "" {
		snap.Stats = append(snap.Stats, Stat{Label: "扣费来源", Value: opendesignFundingLabel(preflight.Funding)})
	}
	snap.Summary = opendesignSummary(snap.Windows, plan != nil && plan.Eligible, plan != nil)
	return snap, nil
}

func opendesignCredits(v opendesignAmount) (float64, bool) {
	text := strings.TrimSpace(string(v))
	if text == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
		return 0, false
	}
	return n, true
}

// opendesignWindowDetail 拿到 credits→USD 比例时显示美元，否则显示原始 credits。
func opendesignWindowDetail(used, limit, creditsPerUSD float64, usedText, limitText opendesignAmount) string {
	if creditsPerUSD > 0 {
		return fmt.Sprintf("%s / %s", opendesignUSDValue(used/creditsPerUSD), opendesignUSDValue(limit/creditsPerUSD))
	}
	return fmt.Sprintf("%s / %s credits", formatCredits(string(usedText)), formatCredits(string(limitText)))
}

func opendesignUSDValue(v float64) string {
	if v > 0 && v < 0.01 {
		return "<$0.01"
	}
	return fmt.Sprintf("$%.2f", v)
}

// formatCredits 为整数 credits 添加千分位；非整数原样返回。
func formatCredits(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return v
		}
	}
	var b strings.Builder
	n := len(v)
	for i := 0; i < n; i++ {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteByte(v[i])
	}
	return b.String()
}

func opendesignWindowMeta(seconds int64) (string, string) {
	switch {
	case seconds == 5*60*60:
		return "five-hour", "5 小时"
	case seconds == 7*24*60*60:
		return "weekly", "每周"
	case seconds > 0 && seconds%(24*60*60) == 0:
		return fmt.Sprintf("window-%ds", seconds), fmt.Sprintf("%d 天", seconds/(24*60*60))
	case seconds > 0 && seconds%3600 == 0:
		return fmt.Sprintf("window-%ds", seconds), fmt.Sprintf("%d 小时", seconds/3600)
	case seconds > 0 && seconds%60 == 0:
		return fmt.Sprintf("window-%ds", seconds), fmt.Sprintf("%d 分钟", seconds/60)
	default:
		return fmt.Sprintf("window-%ds", seconds), "额度窗口"
	}
}

func opendesignTierLabel(tier string) string {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case "go":
		return "Go"
	case "plus":
		return "Go Plus"
	case "pro":
		return "Go Pro"
	case "max":
		return "Go Max"
	case "free":
		return "Free"
	}
	return "OpenDesign"
}

func opendesignFundingLabel(funding string) string {
	switch funding {
	case "coding_plan":
		return "Design / Coding Plan 额度"
	case "wallet":
		return "钱包余额"
	case "gateway":
		return "网关额度"
	}
	return funding
}

func opendesignUSD(v string) string {
	n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return "$" + strings.TrimSpace(v)
	}
	return fmt.Sprintf("$%.2f", n)
}

func opendesignSummary(windows []Window, eligible, hasPlan bool) string {
	if len(windows) == 0 {
		if hasPlan && !eligible {
			return "当前工作区没有生效的 Design / Coding Plan，仅可使用钱包余额"
		}
		return "未返回额度窗口，请确认账号套餐与工作区"
	}
	worst := 0.0
	for _, w := range windows {
		if w.Percent > worst {
			worst = w.Percent
		}
	}
	switch {
	case worst >= 100:
		return "额度窗口已用尽，等待重置或改用钱包余额"
	case worst >= 80:
		return fmt.Sprintf("额度已用 %.0f%%，余量偏低", worst)
	default:
		return fmt.Sprintf("额度充足，最高已用 %.0f%%", worst)
	}
}

func opendesignError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "HTTP 401"), strings.Contains(msg, "HTTP 403"):
		return fmt.Errorf("Control Key 无效或无权访问该工作区，请检查工作区 ID 与 Control Key")
	case strings.Contains(msg, "HTTP 404"):
		return fmt.Errorf("工作区不存在或该账号没有生效的 Design / Coding Plan")
	}
	return err
}

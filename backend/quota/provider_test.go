package quota

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeStatus(t *testing.T) {
	cases := []struct {
		status  string
		percent float64
		want    string
	}{
		{"ok", 10, "ok"},
		{"ok", 85, "warning"},
		{"rate-limited", 40, "exceeded"},
		{"ok", 100, "exceeded"},
	}
	for _, c := range cases {
		if got := normalizeStatus(c.status, c.percent); got != c.want {
			t.Errorf("normalizeStatus(%q, %v) = %q, 期望 %q", c.status, c.percent, got, c.want)
		}
	}
}

func TestLookupCommandcodePlan(t *testing.T) {
	cases := []struct {
		planID string
		name   string
		total  float64
	}{
		{"individual-go", "Go", 10},
		{"individual-goat", "GOAT", 70},
		{"individual-pro", "Pro", 80},
		{"individual-provider", "Provider", 0},
		{"individual-max", "Max", 150},
		{"individual-max-20x", "Max 20x", 300},
		{"individual-pro-v2", "individual-pro-v2", 0},
	}
	for _, c := range cases {
		plan, ok := lookupCommandcodePlan(c.planID)
		if !ok || plan.Name != c.name || plan.Monthly != c.total {
			t.Errorf("lookupCommandcodePlan(%q) = %+v, 期望 %s/%v", c.planID, plan, c.name, c.total)
		}
	}
}

func TestCommandcodeSubscriptionMapping(t *testing.T) {
	requests := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer user_test" || r.Header.Get("x-api-key") != "user_test" {
			t.Errorf("认证头不正确: %s", r.URL.Path)
		}
		requests[r.URL.RequestURI()] = true
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case commandcodePathWhoami:
			_, _ = w.Write([]byte(`{"user":{"userName":"tester"}}`))
		case commandcodePathCredits:
			_, _ = w.Write([]byte(`{"credits":{"monthlyCredits":34.9996,"purchasedCredits":0,"freeCredits":0},"windowLimits":{"fiveHour":{"used":0,"cap":14,"resetAt":0},"weekly":{"used":0,"cap":35,"resetAt":0}}}`))
		case commandcodePathSubs:
			_, _ = w.Write([]byte(`{"data":{"planId":"individual-goat","status":"active","currentPeriodStart":"2026-09-17T07:52:42.000Z","currentPeriodEnd":"2026-10-17T07:52:42.000Z"}}`))
		case commandcodePathSummary:
			_, _ = w.Write([]byte(`{"totalCount":16199,"totalCost":35.0275,"totalCredits":35.0275,"totalMonthlyCredits":35.0275,"periodBasis":"billing-period"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snap, err := queryCommandcode(&QueryContext{Account: &Account{Key: "user_test"}}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Plan != "GOAT" || snap.Status != "active" || snap.Balances[0].Value != "剩余 $35.00 / 合计 $70.03（已用 $35.03）" {
		t.Fatalf("订阅映射不正确: %+v", snap)
	}
	if len(snap.Windows) != 3 || snap.Windows[0].Key != "rolling" || snap.Windows[1].Key != "weekly" || snap.Windows[2].Key != "monthly" {
		t.Fatalf("应显示滚动、每周、月度三个额度窗口: %+v", snap.Windows)
	}
	monthly := snap.Windows[2]
	if monthly.Detail != "$35.03 / $70.03" || monthly.Percent < 50 || monthly.Percent > 50.1 || monthly.ResetAt != formatISODisplay("2026-10-17T07:52:42.000Z") {
		t.Fatalf("月度额度窗口不正确: %+v", monthly)
	}
	if !requests[commandcodePathWhoami+"?limits=1"] || !requests[commandcodePathCredits] || !requests[commandcodePathSubs] || !requests[commandcodePathSummary+"?since=2026-09-17T07%3A52%3A42.000Z"] {
		t.Fatalf("请求路径不正确: %+v", requests)
	}
}

func TestCommandcodeOrganizationScope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != commandcodePathWhoami && r.URL.Query().Get("orgId") != "org/1" {
			t.Errorf("组织范围丢失: %s", r.URL)
		}
		switch r.URL.Path {
		case commandcodePathWhoami:
			_, _ = w.Write([]byte(`{"org":{"id":"org/1","login":"team"}}`))
		case commandcodePathCredits:
			_, _ = w.Write([]byte(`{"credits":{"monthlyCredits":0,"purchasedCredits":12,"freeCredits":0}}`))
		case commandcodePathSubs:
			_, _ = w.Write([]byte(`{"data":{"planId":"individual-provider","status":"active"}}`))
		case commandcodePathSummary:
			_, _ = w.Write([]byte(`{"totalMonthlyCredits":0,"periodBasis":"billing-period"}`))
		}
	}))
	defer server.Close()
	snap, err := queryCommandcode(&QueryContext{Account: &Account{Key: "user_test"}}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Plan != "Provider" || snap.Status != "active" || len(snap.Windows) != 0 || len(snap.Balances) != 3 || strings.Contains(snap.Summary, "月度") {
		t.Fatalf("按量套餐不应显示虚构月度额度: %+v", snap)
	}
}

func TestDeepseekBalanceMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("认证头不正确: %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"is_available":true,"balance_infos":[{"currency":"CNY","total_balance":"474.63","granted_balance":"0.00","topped_up_balance":"474.63"}]}`))
	}))
	defer server.Close()

	snap, err := queryDeepseek(&QueryContext{Account: &Account{Key: "sk-test"}}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Provider != "deepseek" || snap.Status != "ok" || len(snap.Balances) != 1 || snap.Balances[0].Value != "¥474.63" {
		t.Fatalf("DeepSeek 余额映射不正确: %+v", snap)
	}
	if len(snap.Stats) != 2 || snap.Stats[1].Value != "¥474.63" {
		t.Fatalf("DeepSeek 明细不正确: %+v", snap.Stats)
	}
}

// TestMimoPlanUsageMapping 的样例取自 2026-09-26 真实控制台页面
// （https://platform.xiaomimimo.com/console/plan-manage）渲染后的 DOM 抽取结果。
func TestMimoPlanUsageMapping(t *testing.T) {
	payload := `{"status":200,"body":{"planName":"Lite 月度套餐","expiresAt":"2026-10-23 23:59:59 (UTC)","used":"508,596,984","limit":"4,100,000,000","percent":12,"tokenTotal":"45,295,963 Tokens","requestCount":"213 次"}}`
	snap, err := mimoProvider{}.Query(&QueryContext{Account: &Account{ID: 1}, Browser: fakeBrowserRunner{result: payload}})
	if err != nil {
		t.Fatal(err)
	}
	if snap.Plan != "Lite 月度套餐" || snap.Status != "ok" {
		t.Fatalf("MiMo 概览不正确: %+v", snap)
	}
	if len(snap.Windows) != 1 {
		t.Fatalf("MiMo 应有且仅有一个套餐用量窗口: %+v", snap.Windows)
	}
	w := snap.Windows[0]
	if w.Label != "套餐用量" || w.Detail != "508,596,984 / 4,100,000,000" || w.Percent != 12 || w.ResetAt != "2026-10-23 23:59:59 (UTC)" {
		t.Fatalf("MiMo 套餐用量不正确: %+v", w)
	}
	if len(snap.Stats) != 2 || snap.Stats[0].Value != "45,295,963 Tokens" || snap.Stats[1].Value != "213 次" {
		t.Fatalf("MiMo 用量统计不正确: %+v", snap.Stats)
	}
}

func TestMimoEmptyPageReportsError(t *testing.T) {
	_, err := mimoProvider{}.Query(&QueryContext{Account: &Account{ID: 1}, Browser: fakeBrowserRunner{result: `{"status":200,"body":null}`}})
	if err == nil {
		t.Fatal("未登录或页面无数据时应返回错误提示")
	}
}

func TestBuildScrapeRuleScript(t *testing.T) {
	script, err := buildScrapeRuleScript("mimo")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(script, "__RULES__") {
		t.Fatal("规则占位符未被替换")
	}
	for _, want := range []string{"planName", "aria-valuenow", "console/plan-manage"} {
		if !strings.Contains(script, want) {
			t.Fatalf("生成的抓取脚本缺少 %q", want)
		}
	}
	if _, err := buildScrapeRuleScript("not-exist"); err == nil {
		t.Fatal("缺少规则时应报错")
	}
}

type fakeBrowserRunner struct{ result string }

func (f fakeBrowserRunner) Open(uint, string) error                  { return nil }
func (f fakeBrowserRunner) Complete(uint)                            {}
func (f fakeBrowserRunner) Remove(uint) error                        { return nil }
func (f fakeBrowserRunner) Close()                                   {}
func (f fakeBrowserRunner) Run(uint, string, string) (string, error) { return f.result, nil }

func TestMaskKey(t *testing.T) {
	if got := maskKey("user_1234567890abcdef"); got != "user_123...cdef" {
		t.Errorf("maskKey 结果异常: %s", got)
	}
	if got := maskKey("short"); got != "shor..." {
		t.Errorf("maskKey 短 key 结果异常: %s", got)
	}
}

func TestSupportedProviders(t *testing.T) {
	want := []string{"claude-web", "codex-web", "commandcode", "copilot", "cursor-web", "deepseek", "devin", "grok", "kimi", "kimi-code", "mimo", "minimax", "openai-api", "openrouter", "opencode", "zai"}
	for _, id := range want {
		if _, ok := GetProvider(id); !ok {
			t.Fatalf("缺少供应商适配器: %s", id)
		}
	}
}

func TestDevinStatusMapping(t *testing.T) {
	var result devinStatus
	fixture := `{"userStatus":{"planStatus":{"planInfo":{"planName":"Max"},"dailyQuotaRemainingPercent":100,"weeklyQuotaRemainingPercent":"40","overageBalanceMicros":"964220000","dailyQuotaResetAtUnix":"1774080000","weeklyQuotaResetAtUnix":"1774166400"}}}`
	if err := json.Unmarshal([]byte(fixture), &result); err != nil {
		t.Fatal(err)
	}
	snap, err := mapDevinStatus(result)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Plan != "Max" || len(snap.Windows) != 2 || snap.Windows[1].Percent != 60 || len(snap.Balances) != 1 || snap.Balances[0].Value != "$964.22" {
		t.Fatalf("Devin 映射结果异常: %+v", snap)
	}
}

func TestProviderWindowClassification(t *testing.T) {
	for _, tc := range []struct {
		unit, count float64
		want        string
	}{{3, 5, "session"}, {4, 3, "weekly"}, {6, 1, "weekly"}} {
		key, _, ok := zaiWindow(tc.unit, tc.count)
		if !ok || key != tc.want {
			t.Fatalf("Z.ai unit=%v count=%v 得到 %q", tc.unit, tc.count, key)
		}
	}
	if got := weeklyAllowance(1500); got != 150 {
		t.Fatalf("MiniMax weekly boost 得到 %v", got)
	}
	if got := kimiWindowLabel(300, "TIME_UNIT_MINUTE"); got != "5 小时" {
		t.Fatalf("Kimi 窗口标签得到 %q", got)
	}
}

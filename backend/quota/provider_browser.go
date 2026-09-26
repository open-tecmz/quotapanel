package quota

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quotapanel/backend/base/logging"
)

// 这些端点来自站点自身的登录会话，可能随站点更新而变化。
// 在对应站点的浏览器上下文中请求，不导出 Cookie 或访问令牌。
func init() {
	Register(claudeWebProvider{})
	Register(codexWebProvider{})
	Register(cursorWebProvider{})
}

type browserResponse struct {
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func browserFetch(qc *QueryContext, pageURL, script string, out any) error {
	if qc.Browser == nil {
		return fmt.Errorf("浏览器查询不可用")
	}
	raw, err := qc.Browser.Run(qc.Account.ID, pageURL, script)
	if err != nil {
		return err
	}
	var res browserResponse
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return fmt.Errorf("额度响应格式错误: %w", err)
	}
	if res.Status == 401 || res.Status == 403 {
		return fmt.Errorf("登录已过期，请重新登录")
	}
	if res.Status < 200 || res.Status >= 300 {
		return fmt.Errorf("额度查询失败：HTTP %d", res.Status)
	}
	if len(res.Body) == 0 || string(res.Body) == "null" {
		return fmt.Errorf("额度接口未返回数据")
	}
	// 浏览器类供应商的接口没有公开文档，响应结构靠逆向推断，容易随站点改版失效。
	// 这里把原始响应写入日志，便于对照真实结构校正 provider 的字段映射。
	logging.Debug("quota browser raw response: %s", truncateText(strings.Join(strings.Fields(string(res.Body)), ""), 800))
	return json.Unmarshal(res.Body, out)
}

type claudeWebProvider struct{}

func (claudeWebProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "claude-web", Name: "Claude 订阅", Description: "在独立登录窗口登录 Claude 后读取 5 小时及每周额度", Mode: ModeBrowser, DocsURL: "https://github.com/f-is-h/Usage4Claude"}
}
func (claudeWebProvider) LoginURL() string { return "https://claude.ai/settings/usage" }

func (p claudeWebProvider) Query(qc *QueryContext) (*Snapshot, error) {
	const script = `(async()=>{const o=await fetch('/api/organizations',{credentials:'include',cache:'no-store'});if(!o.ok)return JSON.stringify({status:o.status,body:null});const orgs=await o.json();const id=Array.isArray(orgs)&&orgs[0]&&(orgs[0].uuid||orgs[0].id);if(!id)return JSON.stringify({status:404,body:null});const r=await fetch('/api/organizations/'+encodeURIComponent(id)+'/usage',{credentials:'include',cache:'no-store'});return JSON.stringify({status:r.status,body:await r.json()})})()`
	var body map[string]json.RawMessage
	if err := browserFetch(qc, p.LoginURL(), script, &body); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "claude-web", Plan: "Claude 订阅", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	for _, field := range []struct{ Key, Label string }{{"five_hour", "5 小时"}, {"seven_day", "每周"}, {"seven_day_opus", "Opus 每周"}, {"seven_day_sonnet", "Sonnet 每周"}} {
		var v struct {
			Utilization float64 `json:"utilization"`
			ResetsAt    string  `json:"resets_at"`
		}
		if len(body[field.Key]) == 0 || string(body[field.Key]) == "null" {
			continue
		}
		if err := json.Unmarshal(body[field.Key], &v); err != nil {
			continue
		}
		status := normalizeStatus("", v.Utilization)
		if status == "exceeded" {
			snap.Status = "exceeded"
		} else if status == "warning" && snap.Status == "ok" {
			snap.Status = "warning"
		}
		snap.Windows = append(snap.Windows, Window{Key: field.Key, Label: field.Label, Percent: v.Utilization, Status: status, ResetAt: formatISODisplay(v.ResetsAt)})
	}
	if len(snap.Windows) == 0 {
		return nil, fmt.Errorf("Claude 未返回额度窗口，请确认订阅和登录状态")
	}
	return snap, nil
}

type codexWebProvider struct{}

func (codexWebProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "codex-web", Name: "ChatGPT / Codex", Description: "在独立登录窗口登录 ChatGPT 后读取 Codex 订阅使用窗口", Mode: ModeBrowser, DocsURL: "https://github.com/deviffyy/OpenQuota"}
}
func (codexWebProvider) LoginURL() string { return "https://chatgpt.com/" }

func (p codexWebProvider) Query(qc *QueryContext) (*Snapshot, error) {
	const script = `(async()=>{const s=await fetch('/api/auth/session',{credentials:'include',cache:'no-store'});if(!s.ok)return JSON.stringify({status:s.status,body:null});const session=await s.json();if(!session.accessToken)return JSON.stringify({status:401,body:null});const r=await fetch('/backend-api/wham/usage',{credentials:'include',cache:'no-store',headers:{Authorization:'Bearer '+session.accessToken}});return JSON.stringify({status:r.status,body:await r.json()})})()`
	var body struct {
		Plan      string `json:"plan_type"`
		RateLimit struct {
			Primary   *codexWindow `json:"primary_window"`
			Secondary *codexWindow `json:"secondary_window"`
		} `json:"rate_limit"`
	}
	if err := browserFetch(qc, p.LoginURL(), script, &body); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "codex-web", Plan: strings.ToUpper(body.Plan), Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	for i, v := range []*codexWindow{body.RateLimit.Primary, body.RateLimit.Secondary} {
		if v == nil {
			continue
		}
		key, label := "five-hour", "5 小时"
		if i == 1 {
			key, label = "weekly", "每周"
		}
		status := normalizeStatus("", v.UsedPercent)
		if status == "exceeded" {
			snap.Status = "exceeded"
		} else if status == "warning" && snap.Status == "ok" {
			snap.Status = "warning"
		}
		reset := ""
		if v.ResetAt > 0 {
			reset = time.Unix(v.ResetAt, 0).Local().Format("01-02 15:04")
		}
		snap.Windows = append(snap.Windows, Window{Key: key, Label: label, Percent: v.UsedPercent, Status: status, ResetAt: reset})
	}
	if len(snap.Windows) == 0 {
		return nil, fmt.Errorf("Codex 未返回额度窗口，请确认当前账号支持 Codex")
	}
	return snap, nil
}

type codexWindow struct {
	UsedPercent float64 `json:"used_percent"`
	ResetAt     int64   `json:"reset_at"`
}

type cursorWebProvider struct{}

func (cursorWebProvider) Info() ProviderInfo {
	return ProviderInfo{ID: "cursor-web", Name: "Cursor", Description: "在独立登录窗口登录 Cursor 后读取当前周期用量", Mode: ModeBrowser, DocsURL: "https://github.com/deviffyy/OpenQuota"}
}
func (cursorWebProvider) LoginURL() string { return "https://cursor.com/dashboard" }

func (p cursorWebProvider) Query(qc *QueryContext) (*Snapshot, error) {
	const script = `(async()=>{const r=await fetch('/api/usage-summary',{credentials:'include',cache:'no-store'});return JSON.stringify({status:r.status,body:await r.json()})})()`
	var body map[string]any
	if err := browserFetch(qc, p.LoginURL(), script, &body); err != nil {
		return nil, err
	}
	snap := &Snapshot{Provider: "cursor-web", Plan: "Cursor 订阅", Status: "ok", Windows: []Window{}, Balances: []Stat{}, Stats: []Stat{}, UpdatedAt: now()}
	if plan, ok := body["planUsage"].(map[string]any); ok {
		if percent, ok := plan["totalPercentUsed"].(float64); ok {
			percent = max(0, min(100, percent))
			status := normalizeStatus("", percent)
			snap.Status = status
			reset, _ := body["billingCycleEnd"].(string)
			snap.Windows = append(snap.Windows, Window{Key: "period", Label: "当前周期", Percent: percent, Status: status, ResetAt: formatISODisplay(reset)})
		}
	}
	if len(snap.Windows) == 0 {
		if individual, ok := body["individualUsage"].(map[string]any); ok {
			if plan, ok := individual["plan"].(map[string]any); ok {
				if percent, ok := plan["totalPercentUsed"].(float64); ok {
					percent = max(0, min(100, percent))
					status := normalizeStatus("", percent)
					snap.Status = status
					reset, _ := body["billingCycleEnd"].(string)
					snap.Windows = append(snap.Windows, Window{Key: "period", Label: "当前周期", Percent: percent, Status: status, ResetAt: formatISODisplay(reset)})
				}
			}
		}
	}
	if len(snap.Windows) == 0 {
		return nil, fmt.Errorf("Cursor 未返回可识别的额度数据")
	}
	return snap, nil
}

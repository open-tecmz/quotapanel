package quota

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"
)

// ── 迷你面板（macOS 菜单栏弹窗）──────────────────────────────────────────────
// 面板 HTML 由 html/template 渲染（模板见 minipanel.html，构建时嵌入二进制）。
// 模板内含 CSS / JS，配色与 frontend/packages/ui/src/theme/tokens.css 保持一致，
// 并通过 prefers-color-scheme 跟随系统深浅色；改色时同步两侧。

//go:embed minipanel.html
var miniPanelTemplateSource string

// miniPanelTemplate 在包初始化时解析一次，模板语法错误会直接暴露在测试 / 启动阶段。
var miniPanelTemplate = template.Must(template.New("minipanel").Parse(miniPanelTemplateSource))

// miniMaxWindows 是单个账号最多展示的额度窗口条数，超出折叠为「还有 N 项」。
const miniMaxWindows = 3

// ── 模板视图模型 ─────────────────────────────────────────────────────────────

type miniPanelView struct {
	Lang        string
	Refresh     string
	Open        string
	Empty       string
	EmptyHint   string
	AutoRefresh string
	UpdatedText string
	Items       []miniPanelItemView
}

type miniPanelItemView struct {
	ID         uint
	Name       string
	Meta       string
	Badge      string
	BadgeClass string
	Plan       string
	Error      string
	Empty      string
	More       string
	Windows    []miniPanelWindowView
	Stats      []miniPanelStatView
}

type miniPanelWindowView struct {
	Label string
	Width string
	Class string
	Value string
}

type miniPanelStatView struct {
	Label string
	Value string
	Muted bool
}

// ── 本地化文案 ───────────────────────────────────────────────────────────────

// miniTexts 是迷你面板的本地化文案。
type miniTexts struct {
	Refresh        string
	Open           string
	Empty          string
	EmptyHint      string
	NotQueried     string
	UpdatedPrefix  string
	AutoRefresh    string
	StatusOk       string
	StatusWarning  string
	StatusExceeded string
	StatusUnknown  string
	More           string // 含一个 %d 占位符
}

func miniTextsFor(locale string) miniTexts {
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return miniTexts{
			Refresh:        "Refresh",
			Open:           "Open",
			Empty:          "No subscriptions yet",
			EmptyHint:      "Open the app to add your first account",
			NotQueried:     "Not queried",
			UpdatedPrefix:  "Updated ",
			AutoRefresh:    "Auto-refresh every 5 min",
			StatusOk:       "OK",
			StatusWarning:  "Low",
			StatusExceeded: "Exceeded",
			StatusUnknown:  "Unknown",
			More:           "%d more",
		}
	}
	return miniTexts{
		Refresh:        "刷新",
		Open:           "打开界面",
		Empty:          "还没有添加订阅",
		EmptyHint:      "点击「打开界面」添加你的第一个订阅账号",
		NotQueried:     "未查询",
		UpdatedPrefix:  "更新于 ",
		AutoRefresh:    "每 5 分钟自动刷新",
		StatusOk:       "正常",
		StatusWarning:  "即将用尽",
		StatusExceeded: "已超限",
		StatusUnknown:  "未知",
		More:           "还有 %d 项",
	}
}

// ── 渲染 ─────────────────────────────────────────────────────────────────────

// BuildMiniPanelHTML 生成菜单栏迷你面板的完整 HTML 文档。
// accounts 决定展示顺序，snaps 为按账号 ID 索引的最新额度快照。
func BuildMiniPanelHTML(accounts []AccountView, snaps map[uint]*Snapshot, locale string) string {
	tx := miniTextsFor(locale)
	view := miniPanelView{
		Lang:        langCode(locale),
		Refresh:     tx.Refresh,
		Open:        tx.Open,
		Empty:       tx.Empty,
		EmptyHint:   tx.EmptyHint,
		AutoRefresh: tx.AutoRefresh,
		Items:       make([]miniPanelItemView, 0, len(accounts)),
	}
	if latest := latestUpdateText(snaps); latest != "" {
		view.UpdatedText = tx.UpdatedPrefix + latest
	}
	for i := range accounts {
		view.Items = append(view.Items, miniPanelItemViewFor(accounts[i], snaps[accounts[i].ID], tx))
	}

	var buf bytes.Buffer
	if err := miniPanelTemplate.Execute(&buf, view); err != nil {
		return fallbackMiniPanelHTML(err)
	}
	return buf.String()
}

func miniPanelItemViewFor(acc AccountView, snap *Snapshot, tx miniTexts) miniPanelItemView {
	meta := acc.ProviderName
	if acc.Note != "" {
		meta = acc.Note + " · " + meta
	}
	item := miniPanelItemView{
		ID:    acc.ID,
		Name:  acc.Name,
		Meta:  meta,
		Empty: tx.NotQueried,
	}
	item.Badge, item.BadgeClass = miniStatusBadge(snap, tx)

	switch {
	case snap == nil:
		return item
	case snap.Error != "":
		item.Error = snap.Error
		return item
	}

	item.Plan = snap.Plan
	if len(snap.Windows) > 0 {
		for _, w := range snap.Windows[:min(miniMaxWindows, len(snap.Windows))] {
			item.Windows = append(item.Windows, miniWindowViewFor(w))
		}
		if extra := len(snap.Windows) - miniMaxWindows; extra > 0 {
			item.More = fmt.Sprintf(tx.More, extra)
		}
		return item
	}

	rows := make([]Stat, 0, len(snap.Balances)+len(snap.Stats))
	rows = append(rows, snap.Balances...)
	rows = append(rows, snap.Stats...)
	for _, s := range rows[:min(miniMaxWindows, len(rows))] {
		item.Stats = append(item.Stats, miniPanelStatView{Label: s.Label, Value: s.Value, Muted: s.Muted})
	}
	if extra := len(rows) - miniMaxWindows; extra > 0 {
		item.More = fmt.Sprintf(tx.More, extra)
	}
	return item
}

func miniWindowViewFor(w Window) miniPanelWindowView {
	percent := w.Percent
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	value := w.Detail
	if value == "" {
		value = fmt.Sprintf("%.0f%%", w.Percent)
	}
	return miniPanelWindowView{
		Label: w.Label,
		Width: strconv.FormatFloat(percent, 'f', 1, 64),
		Class: miniProgressClass(w.Status),
		Value: value,
	}
}

// miniStatusBadge 返回账号状态徽章的文字与样式类。
func miniStatusBadge(snap *Snapshot, tx miniTexts) (text string, class string) {
	if snap == nil {
		return tx.NotQueried, "neutral"
	}
	status := snap.Status
	if snap.Error != "" {
		status = "exceeded"
	}
	switch status {
	case "ok", "active":
		return tx.StatusOk, "ok"
	case "warning":
		return tx.StatusWarning, "warning"
	case "exceeded":
		return tx.StatusExceeded, "danger"
	default:
		return tx.StatusUnknown, "neutral"
	}
}

func miniProgressClass(status string) string {
	switch status {
	case "ok":
		return "ok"
	case "warning":
		return "warning"
	case "exceeded":
		return "danger"
	default:
		return "muted"
	}
}

// fallbackMiniPanelHTML 在模板渲染异常时兜底，避免面板空白。
func fallbackMiniPanelHTML(err error) string {
	return "<!DOCTYPE html><html><head><meta charset=\"utf-8\"><style>" +
		"body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;font-size:12px;" +
		"background:#ffffff;color:#0f172a;margin:0;padding:16px}" +
		"@media (prefers-color-scheme:dark){body{background:#111827;color:#f3f4f6}}" +
		"</style></head><body>" + template.HTMLEscapeString("面板渲染失败："+err.Error()) + "</body></html>"
}

// MiniPanelHeight 估算面板初始高度（px），仅用于弹窗首帧尺寸；页面加载后会自行校正。
func MiniPanelHeight(accounts []AccountView, snaps map[uint]*Snapshot) int {
	if len(accounts) == 0 {
		return 168
	}
	height := 92 // 顶栏 + 底栏 + 内边距
	for i := range accounts {
		height += 74 // 名称 + 来源 + 间距
		snap := snaps[accounts[i].ID]
		if snap == nil {
			continue
		}
		if snap.Plan != "" {
			height += 18
		}
		if snap.Error != "" {
			height += 44
			continue
		}
		rows := len(snap.Windows)
		if rows == 0 {
			rows = len(snap.Balances) + len(snap.Stats)
		}
		if rows > miniMaxWindows {
			rows = miniMaxWindows
		}
		height += rows * 22
	}
	if height < 160 {
		height = 160
	}
	if height > 620 {
		height = 620
	}
	return height
}

func langCode(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return "en"
	}
	return "zh"
}

func latestUpdateText(snaps map[uint]*Snapshot) string {
	var newest time.Time
	for _, s := range snaps {
		if s != nil && s.UpdatedAt.After(newest) {
			newest = s.UpdatedAt
		}
	}
	if newest.IsZero() {
		return ""
	}
	return newest.Format("15:04")
}

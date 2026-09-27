package quota

import (
	"strings"
	"testing"
	"time"
)

func testAccounts() []AccountView {
	return []AccountView{
		{ID: 1, Provider: "opencode", ProviderName: "OpenCode", Name: "我的 OpenCode", Note: "主力"},
		{ID: 2, Provider: "deepseek", ProviderName: "DeepSeek API", Name: "DeepSeek"},
	}
}

func testSnaps() map[uint]*Snapshot {
	return map[uint]*Snapshot{
		1: {
			AccountID: 1,
			Plan:      "Pro",
			Status:    "ok",
			Windows: []Window{
				{Key: "5h", Label: "5 小时", Percent: 42, Status: "ok", Detail: "42%"},
				{Key: "week", Label: "每周", Percent: 88, Status: "warning"},
				{Key: "month", Label: "每月", Percent: 10, Status: "ok"},
				{Key: "extra", Label: "额外", Percent: 5, Status: "ok"},
			},
			UpdatedAt: time.Now(),
		},
		2: {
			AccountID: 2,
			Status:    "exceeded",
			Balances:  []Stat{{Label: "余额", Value: "$1.16"}},
			UpdatedAt: time.Now(),
		},
	}
}

func TestBuildMiniPanelHTMLRendersAccounts(t *testing.T) {
	html := BuildMiniPanelHTML(testAccounts(), testSnaps(), "zh")

	for _, want := range []string{
		"我的 OpenCode",
		"OpenCode",
		"DeepSeek API",
		"正常",  // status ok
		"已超限", // status exceeded
		"5 小时",
		"42%",
		"还有 1 项", // 4 windows → 3 shown + 1 more
		"$1.16",
		"刷新",
		"打开界面",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("面板 HTML 缺少 %q", want)
		}
	}
	if strings.Contains(html, "还有 2 项") {
		t.Error("多余窗口统计不应重复计算")
	}
}

func TestBuildMiniPanelHTMLProgressClass(t *testing.T) {
	html := BuildMiniPanelHTML(testAccounts(), testSnaps(), "zh")
	if !strings.Contains(html, `class="fill warning" style="width:88.0%"`) {
		t.Error("警告态进度条应使用 warning 类并按百分比设置宽度")
	}
	if !strings.Contains(html, `class="fill ok" style="width:42.0%"`) {
		t.Error("正常态进度条应使用 ok 类")
	}
}

func TestBuildMiniPanelHTMLEscapesUserContent(t *testing.T) {
	accounts := []AccountView{{ID: 7, Provider: "x", ProviderName: "<b>厂商</b>", Name: `<script>alert(1)</script>`}}
	snaps := map[uint]*Snapshot{7: {AccountID: 7, Status: "exceeded", Error: "<img src=x>"}}

	html := BuildMiniPanelHTML(accounts, snaps, "zh")
	if strings.Contains(html, "<script>alert(1)</script>") {
		t.Error("账号名中的脚本标签应被转义")
	}
	if strings.Contains(html, "<img src=x>") {
		t.Error("错误信息中的标签应被转义")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("转义后的账号名未出现在面板中")
	}
}

func TestBuildMiniPanelHTMLEmptyAndEnglish(t *testing.T) {
	empty := BuildMiniPanelHTML(nil, nil, "zh")
	if !strings.Contains(empty, "还没有添加订阅") {
		t.Error("无账号时应展示空状态")
	}

	en := BuildMiniPanelHTML(testAccounts(), testSnaps(), "en")
	for _, want := range []string{"Refresh", "Open", "OK", "Exceeded", "1 more"} {
		if !strings.Contains(en, want) {
			t.Errorf("英文面板缺少 %q", want)
		}
	}
}

func TestMiniPanelHeightBounds(t *testing.T) {
	if got := MiniPanelHeight(nil, nil); got < 160 || got > 620 {
		t.Errorf("空列表高度越界: %d", got)
	}
	snaps := testSnaps()
	small := MiniPanelHeight(testAccounts()[:1], snaps)
	if small <= 0 {
		t.Errorf("高度估算应大于 0，实际 %d", small)
	}
	large := MiniPanelHeight([]AccountView{
		{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}, {ID: 6}, {ID: 7},
	}, snaps)
	if large > 620 {
		t.Errorf("高度估算应被上限截断，实际 %d", large)
	}
}

package quota

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseOpenDesignCredential(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		workspace string
		token     string
		wantErr   bool
	}{
		{"组合凭据", " ws_1 | vela_ctrl_abc ", "ws_1", "vela_ctrl_abc", false},
		{"仅 Control Key", "vela_ctrl_abc", "", "vela_ctrl_abc", false},
		{"Bearer 前缀", "ws_1|Bearer vela_ctrl_abc", "ws_1", "vela_ctrl_abc", false},
		{"推理 Key", "sk-amr-abc123", "", "", true},
		{"空值", "   ", "", "", true},
		{"非法工作区", "ws 1|vela_ctrl_abc", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cred, err := parseOpenDesignCredential(c.raw)
			if c.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际得到 %+v", cred)
				}
				return
			}
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}
			if cred.workspaceID != c.workspace || cred.token != c.token {
				t.Fatalf("解析结果 %+v，期望 %s/%s", cred, c.workspace, c.token)
			}
		})
	}
}

func TestOpenDesignSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer vela_ctrl_test" {
			t.Errorf("认证头不正确: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("x-vela-workspace-id") != "ws_1" {
			t.Errorf("工作区头不正确: %s", r.Header.Get("x-vela-workspace-id"))
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/workspaces/ws_1/billing/preflight":
			_, _ = w.Write([]byte(`{"schemaVersion":1,"workspaceId":"ws_1","workspaceMemberId":"m_1","balanceUsd":"1.5000","funding":"coding_plan","codingPlan":{"workspaceId":"ws_1","eligible":true,"tier":"go","windows":[{"policyId":"p_week","durationSeconds":604800,"resetMode":"anchored_recurring","usedCredits":"15000","limitCredits":"50000","remainingCredits":"35000","windowStart":"2026-10-17T06:06:54.255Z","resetsAt":"2026-10-24T06:06:54.255Z"},{"policyId":"p_5h","durationSeconds":18000,"resetMode":"anchored_recurring","usedCredits":"5000","limitCredits":"10000","remainingCredits":"5000","windowStart":"2026-10-17T02:06:54.255Z","resetsAt":"2026-10-17T07:06:54.255Z"}]}}`))
		case "/workspaces/ws_1/wallet/balance":
			_, _ = w.Write([]byte(`{"billingScopeVersion":2,"workspaceId":"ws_1","workspaceMemberId":"m_1","balanceUsd":"1.50"}`))
		case "/billing/summary":
			_, _ = w.Write([]byte(`{"balanceUsd":"1.50","creditsPerUsd":10000,"membershipTier":"go"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snap, err := queryOpenDesign(&QueryContext{Account: &Account{Key: "ws_1|vela_ctrl_test"}}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Plan != "Go" || snap.Status != "ok" {
		t.Fatalf("套餐或状态不正确: %+v", snap)
	}
	if len(snap.Windows) != 2 {
		t.Fatalf("窗口数量 %d，期望 2: %+v", len(snap.Windows), snap.Windows)
	}
	if snap.Windows[0].Key != "five-hour" || snap.Windows[0].Percent != 50 || snap.Windows[0].Detail != "$0.50 / $1.00" {
		t.Fatalf("5 小时窗口映射不正确: %+v", snap.Windows[0])
	}
	if snap.Windows[1].Key != "weekly" || snap.Windows[1].Percent != 30 || snap.Windows[1].ResetAt == "-" {
		t.Fatalf("每周窗口映射不正确: %+v", snap.Windows[1])
	}
	if len(snap.Balances) != 1 || snap.Balances[0].Value != "$1.50" {
		t.Fatalf("钱包余额映射不正确: %+v", snap.Balances)
	}
	if len(snap.Stats) != 1 || snap.Stats[0].Value != "Design / Coding Plan 额度" {
		t.Fatalf("扣费来源映射不正确: %+v", snap.Stats)
	}
	if !strings.Contains(snap.Summary, "充足") {
		t.Fatalf("摘要不正确: %s", snap.Summary)
	}
}

func TestOpenDesignWalletFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/workspaces/ws_1/billing/preflight":
			_, _ = w.Write([]byte(`{"workspaceId":"ws_1","workspaceMemberId":"m_1","balanceUsd":"2.00","funding":"wallet","codingPlan":{"workspaceId":"ws_1","eligible":false,"tier":null,"windows":[]}}`))
		case "/workspaces/ws_1/wallet/balance":
			http.Error(w, `{"error":"boom"}`, http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snap, err := queryOpenDesign(&QueryContext{Account: &Account{Key: "ws_1|vela_ctrl_test"}}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Windows) != 0 {
		t.Fatalf("无生效套餐时不应有窗口: %+v", snap.Windows)
	}
	if len(snap.Balances) != 1 || snap.Balances[0].Value != "$2.00" {
		t.Fatalf("钱包查询失败时应回退 preflight 余额: %+v", snap.Balances)
	}
	if !strings.Contains(snap.Summary, "没有生效") {
		t.Fatalf("摘要不正确: %s", snap.Summary)
	}
}

func TestOpenDesignWorkspaceDiscovery(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/workspaces/current":
			_, _ = w.Write([]byte(`{"workspaceId":"ws_current","activeWorkspaceId":"ws_current"}`))
		case "/workspaces/ws_current/billing/preflight":
			_, _ = w.Write([]byte(`{"workspaceId":"ws_current","balanceUsd":"0.50","codingPlan":{"workspaceId":"ws_current","eligible":true,"tier":"plus","windows":[{"policyId":"p","durationSeconds":604800,"usedCredits":"100","limitCredits":"1000","remainingCredits":"900","resetsAt":"2026-11-17T06:06:54.255Z"}]}}`))
		case "/workspaces/ws_current/wallet/balance":
			_, _ = w.Write([]byte(`{"billingScopeVersion":2,"workspaceId":"ws_current","workspaceMemberId":"m_1","balanceUsd":"0.50"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snap, err := queryOpenDesign(&QueryContext{Account: &Account{Key: "vela_ctrl_test"}}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 || paths[0] != "/workspaces/current" {
		t.Fatalf("应先读取当前工作区: %v", paths)
	}
	if snap.Plan != "Go Plus" || len(snap.Windows) != 1 || snap.Windows[0].Percent != 10 {
		t.Fatalf("自动工作区下的映射不正确: %+v", snap)
	}
	// 没有换算比例时退回 credits 文案。
	if snap.Windows[0].Detail != "100 / 1,000 credits" {
		t.Fatalf("credits 文案不正确: %s", snap.Windows[0].Detail)
	}
}

func TestOpenDesignUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := queryOpenDesign(&QueryContext{Account: &Account{Key: "ws_1|vela_ctrl_test"}}, server.URL)
	if err == nil || !strings.Contains(err.Error(), "Control Key") {
		t.Fatalf("401 未给出可读提示: %v", err)
	}
}

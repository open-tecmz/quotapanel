package quota

import "testing"

type fixtureBrowser struct{ response string }

func (f fixtureBrowser) Open(uint, string) error                  { return nil }
func (f fixtureBrowser) Complete(uint)                            {}
func (f fixtureBrowser) Remove(uint) error                        { return nil }
func (f fixtureBrowser) Close()                                   {}
func (f fixtureBrowser) Run(uint, string, string) (string, error) { return f.response, nil }

func TestBrowserProviderMappings(t *testing.T) {
	cases := []struct {
		name        string
		provider    Provider
		response    string
		wantWindows int
		wantStatus  string
	}{
		{"Claude", claudeWebProvider{}, `{"status":200,"body":{"five_hour":{"utilization":82.5,"resets_at":"2026-09-24T10:00:00Z"},"seven_day":{"utilization":20}}}`, 2, "warning"},
		{"Codex", codexWebProvider{}, `{"status":200,"body":{"plan_type":"plus","rate_limit":{"primary_window":{"used_percent":25,"reset_at":1790000000},"secondary_window":{"used_percent":100,"reset_at":1790500000}}}}`, 2, "exceeded"},
		{"Cursor", cursorWebProvider{}, `{"status":200,"body":{"individualUsage":{"plan":{"totalPercentUsed":32}},"billingCycleEnd":"2026-10-01T00:00:00Z"}}`, 1, "ok"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap, err := tc.provider.Query(&QueryContext{Account: &Account{ID: 42}, Browser: fixtureBrowser{tc.response}})
			if err != nil {
				t.Fatal(err)
			}
			if len(snap.Windows) != tc.wantWindows || snap.Status != tc.wantStatus {
				t.Fatalf("快照错误: %+v", snap)
			}
		})
	}
}

func TestBrowserProviderExpiredLogin(t *testing.T) {
	_, err := codexWebProvider{}.Query(&QueryContext{Account: &Account{ID: 1}, Browser: fixtureBrowser{`{"status":401,"body":null}`}})
	if err == nil {
		t.Fatal("过期会话未提示重新登录")
	}
}

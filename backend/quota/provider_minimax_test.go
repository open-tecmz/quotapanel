package quota

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// minimaxGeneralPayload is the shape returned by /v1/token_plan/remains: one
// entry per model class, `general` being the coding plan we display.
const minimaxGeneralPayload = `{"base_resp":{"status_code":0,"status_msg":"success"},"model_remains":[{"model_name":"general","current_interval_remaining_percent":98,"current_weekly_remaining_percent":95,"current_interval_status":1,"current_weekly_status":1,"end_time":1780329600000,"weekly_end_time":1780848000000},{"model_name":"video","current_interval_remaining_percent":100,"current_weekly_remaining_percent":100,"current_interval_status":3,"current_weekly_status":1}]}`

func minimaxServer(t *testing.T, body string) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, &hits
}

// TestMiniMaxChinaKeyProbesNextRegion reproduces a mainland China Token Plan
// key being rejected by the global deployment (base_resp 2049 "invalid api
// key"): the query must move on to the China deployment instead of failing.
func TestMiniMaxChinaKeyProbesNextRegion(t *testing.T) {
	global, globalHits := minimaxServer(t, `{"base_resp":{"status_code":2049,"status_msg":"invalid api key"}}`)
	china, chinaHits := minimaxServer(t, minimaxGeneralPayload)
	useMinimaxEndpoints(t, global.URL, china.URL)

	snap, err := minimaxProvider{}.Query(&QueryContext{Account: &Account{Key: "sk-cp-cn"}})
	if err != nil {
		t.Fatalf("国内 Key 应回退到国内区域: %v", err)
	}
	if globalHits.Load() != 1 || chinaHits.Load() != 1 {
		t.Fatalf("区域探测次数不正确: global=%d china=%d", globalHits.Load(), chinaHits.Load())
	}
	if snap.Provider != "minimax" || snap.Status != "ok" || len(snap.Windows) != 2 {
		t.Fatalf("额度映射不正确: %+v", snap)
	}
	if snap.Windows[0].Key != "session" || snap.Windows[0].Percent != 2 || snap.Windows[1].Key != "weekly" || snap.Windows[1].Percent != 5 {
		t.Fatalf("额度窗口不正确: %+v", snap.Windows)
	}
}

// useMinimaxEndpoints points the provider at local servers for one test.
func useMinimaxEndpoints(t *testing.T, endpoints ...string) {
	t.Helper()
	old := minimaxEndpoints
	minimaxEndpoints = endpoints
	t.Cleanup(func() { minimaxEndpoints = old })
}

// TestMiniMaxRejectedKeyReportsAuthError keeps a key that no region accepts
// reported as an auth failure rather than a generic parse error.
func TestMiniMaxRejectedKeyReportsAuthError(t *testing.T) {
	global, _ := minimaxServer(t, `{"base_resp":{"status_code":2049,"status_msg":"invalid api key"}}`)
	china, _ := minimaxServer(t, `{"base_resp":{"status_code":2049,"status_msg":"invalid api key"}}`)
	useMinimaxEndpoints(t, global.URL, china.URL)

	_, err := minimaxProvider{}.Query(&QueryContext{Account: &Account{Key: "sk-bogus"}})
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("两个区域都拒绝时应报认证错误，实际: %v", err)
	}
}

// TestMiniMaxAcceptedKeyDoesNotProbeOtherRegion guards against masking a usable
// but unexpected response with a second region's auth error.
func TestMiniMaxAcceptedKeyDoesNotProbeOtherRegion(t *testing.T) {
	global, _ := minimaxServer(t, `{"base_resp":{"status_code":0,"status_msg":"success"},"model_remains":[]}`)
	china, chinaHits := minimaxServer(t, minimaxGeneralPayload)
	useMinimaxEndpoints(t, global.URL, china.URL)

	_, err := minimaxProvider{}.Query(&QueryContext{Account: &Account{Key: "sk-key"}})
	if err == nil || !strings.Contains(err.Error(), "未返回 Token Plan 用量") {
		t.Fatalf("Key 已被接受但无用量时应直接报错，实际: %v", err)
	}
	if chinaHits.Load() != 0 {
		t.Fatalf("Key 已被接受时不应再探测其他区域，实际探测 %d 次", chinaHits.Load())
	}
}

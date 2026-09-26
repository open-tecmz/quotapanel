package llmpx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"quotapanel/backend/base/logging"
)

// LLMPXModelInfo 表示 llmpx 平台提供的一个模型。
type LLMPXModelInfo struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Rate float64 `json:"rate"`
}

// VipInfo 表示用户的 VIP 状态。isDefault=true 表示未开通，false 表示已开通。
type VipInfo struct {
	ID        int    `json:"id"`
	Flag      string `json:"flag"`
	Title     string `json:"title"`
	IsDefault bool   `json:"isDefault"`
	Icon      string `json:"icon"`
}

// LLMPXInfo 聚合了模型列表、额度和调用信息，一次性返回给前端。
type LLMPXInfo struct {
	Quota  int              `json:"quota"`
	ApiUrl string           `json:"apiUrl"`
	ApiKey string           `json:"apiKey"`
	Models []LLMPXModelInfo `json:"models"`
	Vip    *VipInfo         `json:"vip"`
}

// FetchInfo 通过 api-token 调用 /app_manager/user_info 获取 llmpx 信息。
func FetchInfo(apiBaseURL, apiToken string) (*LLMPXInfo, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("未登录")
	}
	apiBaseURL = strings.TrimRight(apiBaseURL, "/")
	url := apiBaseURL + "/app_manager/user_info"
	logging.Info("[llmpx] FetchInfo 请求: url=%s, token=%s...", url, apiToken[:min(len(apiToken), 8)])

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBufferString("{}"))
	if err != nil {
		logging.Error("[llmpx] 创建请求失败: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-token", apiToken)

	resp, err := client.Do(req)
	if err != nil {
		logging.Error("[llmpx] 请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	logging.Info("[llmpx] 响应状态: %d, body: %s", resp.StatusCode, string(bodyBytes[:min(len(bodyBytes), 500)]))

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Data struct {
				Llmpx *LLMPXInfo `json:"llmpx"`
				Vip   *VipInfo   `json:"vip"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		logging.Error("[llmpx] 解析响应失败: %v, body: %s", err, string(bodyBytes[:min(len(bodyBytes), 200)]))
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 0 {
		msg := result.Msg
		if msg == "" {
			msg = fmt.Sprintf("API error code: %d", result.Code)
		}
		logging.Error("[llmpx] API 返回错误: code=%d, msg=%s", result.Code, msg)
		return nil, fmt.Errorf("%s", msg)
	}

	if result.Data.Data.Llmpx == nil {
		logging.Info("[llmpx] 响应中无 llmpx 字段，可能未开启")
		info := &LLMPXInfo{Models: []LLMPXModelInfo{}}
		if result.Data.Data.Vip != nil {
			info.Vip = result.Data.Data.Vip
		} else {
			info.Vip = &VipInfo{IsDefault: true}
		}
		return info, nil
	}

	info := result.Data.Data.Llmpx
	if info.Vip == nil {
		if result.Data.Data.Vip != nil {
			info.Vip = result.Data.Data.Vip
		} else {
			info.Vip = &VipInfo{IsDefault: true}
		}
	}
	logging.Info("[llmpx] 成功获取 %d 个模型, quota=%d, apiUrl=%s, vip.isDefault=%v",
		len(info.Models), info.Quota, info.ApiUrl, info.Vip.IsDefault)
	return info, nil
}

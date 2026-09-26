package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"quotapanel/backend/base/config"
	"quotapanel/backend/base/llmpx"
	"quotapanel/backend/base/logging"
)

// CallResult holds the result of an LLM API call.
type CallResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// CallOpenAICompat calls an OpenAI-compatible LLM API (supports openai and gemini providers).
func CallOpenAICompat(client *http.Client, cfg *config.LLMConfig, system, user string) (CallResult, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		switch cfg.Provider {
		case "gemini":
			baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}
	// 确保 baseURL 以 /v1 结尾（兼容只给了域名的情况）
	trimmed := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(trimmed, "/v1") && !strings.Contains(trimmed, "/v1beta") {
		trimmed += "/v1"
	}
	endpoint := trimmed + "/chat/completions"

	messages := []map[string]string{}
	if system != "" {
		messages = append(messages, map[string]string{"role": "system", "content": system})
	}
	messages = append(messages, map[string]string{"role": "user", "content": user})

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CallResult{Error: err.Error()}, err
	}
	if result.Error.Message != "" {
		return CallResult{Error: result.Error.Message}, fmt.Errorf("%s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return CallResult{Error: "empty response"}, fmt.Errorf("empty response from LLM")
	}
	return CallResult{Content: result.Choices[0].Message.Content}, nil
}

// CallOpenAICompatVision calls an OpenAI-compatible vision API with a base64-encoded image.
func CallOpenAICompatVision(client *http.Client, cfg *config.LLMConfig, system, user, imageBase64 string) (CallResult, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		switch cfg.Provider {
		case "gemini":
			baseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
		default:
			baseURL = "https://api.openai.com/v1"
		}
	}
	trimmed := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(trimmed, "/v1") && !strings.Contains(trimmed, "/v1beta") {
		trimmed += "/v1"
	}
	endpoint := trimmed + "/chat/completions"

	userContent := []interface{}{
		map[string]interface{}{"type": "text", "text": user},
		map[string]interface{}{
			"type":      "image_url",
			"image_url": map[string]string{"url": "data:image/png;base64," + imageBase64},
		},
	}

	messages := []map[string]interface{}{}
	if system != "" {
		messages = append(messages, map[string]interface{}{"role": "system", "content": system})
	}
	messages = append(messages, map[string]interface{}{"role": "user", "content": userContent})

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": messages,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CallResult{Error: err.Error()}, err
	}
	if result.Error.Message != "" {
		return CallResult{Error: result.Error.Message}, fmt.Errorf("%s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return CallResult{Error: "empty response"}, fmt.Errorf("empty response from LLM")
	}
	return CallResult{Content: result.Choices[0].Message.Content}, nil
}

// CallClaude calls the Anthropic Claude API.
func CallClaude(client *http.Client, cfg *config.LLMConfig, system, user string) (CallResult, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/messages"

	body := map[string]interface{}{
		"model":      cfg.Model,
		"max_tokens": 4096,
		"messages":   []map[string]string{{"role": "user", "content": user}},
	}
	if system != "" {
		body["system"] = system
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CallResult{Error: err.Error()}, err
	}
	if result.Error.Message != "" {
		return CallResult{Error: result.Error.Message}, fmt.Errorf("%s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return CallResult{Error: "empty response"}, fmt.Errorf("empty response from Claude")
	}
	return CallResult{Content: result.Content[0].Text}, nil
}

// CallClaudeVision calls the Anthropic Claude vision API with a base64-encoded image.
func CallClaudeVision(client *http.Client, cfg *config.LLMConfig, system, user, imageBase64 string) (CallResult, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v1/messages"

	userContent := []interface{}{
		map[string]interface{}{
			"type": "image",
			"source": map[string]string{
				"type":       "base64",
				"media_type": "image/png",
				"data":       imageBase64,
			},
		},
		map[string]interface{}{"type": "text", "text": user},
	}

	body := map[string]interface{}{
		"model":      cfg.Model,
		"max_tokens": 4096,
		"messages":   []map[string]interface{}{{"role": "user", "content": userContent}},
	}
	if system != "" {
		body["system"] = system
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(data))
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return CallResult{Error: err.Error()}, err
	}
	defer resp.Body.Close()

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return CallResult{Error: err.Error()}, err
	}
	if result.Error.Message != "" {
		return CallResult{Error: result.Error.Message}, fmt.Errorf("%s", result.Error.Message)
	}
	if len(result.Content) == 0 {
		return CallResult{Error: "empty response"}, fmt.Errorf("empty response from Claude")
	}
	return CallResult{Content: result.Content[0].Text}, nil
}

// TestConfig 测试大模型配置是否可用。
func TestConfig(provider, apiKey, baseURL, model string) CallResult {
	cfg := &config.LLMConfig{
		Provider: provider,
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
	}
	client := &http.Client{Timeout: 15 * time.Second}
	if cfg.Provider == "claude" {
		result, _ := CallClaude(client, cfg, "You are a helpful assistant.", "Reply with 'OK' only.")
		return result
	}
	result, _ := CallOpenAICompat(client, cfg, "You are a helpful assistant.", "Reply with 'OK' only.")
	return result
}

// resolveBuiltinConfig 解析内置模型，返回对应的 LLMConfig。
func resolveBuiltinConfig(modelName, apiBaseURL, apiToken string) (*config.LLMConfig, error) {
	info, err := llmpx.FetchInfo(apiBaseURL, apiToken)
	if err != nil {
		return nil, fmt.Errorf("获取内置模型信息失败: %w", err)
	}
	if info.ApiUrl == "" || info.ApiKey == "" {
		return nil, fmt.Errorf("内置模型未配置")
	}
	return &config.LLMConfig{
		Provider: "openai",
		APIKey:   info.ApiKey,
		BaseURL:  info.ApiUrl,
		Model:    modelName,
	}, nil
}

// resolveConfig 根据 configID 从配置列表中找到对应的 LLMConfig。
func resolveConfig(configID string, configs []config.LLMConfig) *config.LLMConfig {
	for i := range configs {
		c := &configs[i]
		if configID == "" {
			if c.IsDefault {
				return c
			}
		} else if c.ID == configID {
			return c
		}
	}
	if len(configs) > 0 {
		return &configs[0]
	}
	return nil
}

// Call 调用指定 LLM 配置。
// configID 为 "builtin:modelName" 时使用内置模型，否则从 configs 列表中查找。
func Call(configID string, configs []config.LLMConfig, apiBaseURL, apiToken, systemPrompt, userPrompt string) (CallResult, error) {
	if strings.HasPrefix(configID, "builtin:") {
		modelName := strings.TrimPrefix(configID, "builtin:")
		cfg, err := resolveBuiltinConfig(modelName, apiBaseURL, apiToken)
		if err != nil {
			return CallResult{Error: err.Error()}, err
		}
		client := &http.Client{Timeout: 60 * time.Second}
		logging.Info("[LLM] 调用内置模型 model=%s apiUrl=%s", modelName, cfg.BaseURL)
		result, callErr := CallOpenAICompat(client, cfg, systemPrompt, userPrompt)
		if result.Error != "" {
			logging.Error("[LLM] 内置模型调用失败: %s", result.Error)
		} else {
			logging.Info("[LLM] 内置模型调用成功，响应长度=%d", len(result.Content))
		}
		return result, callErr
	}

	llmCfg := resolveConfig(configID, configs)
	if llmCfg == nil {
		return CallResult{Error: "no LLM config found"}, fmt.Errorf("no LLM config found")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	logging.Info("[LLM] 调用模型 provider=%s model=%s", llmCfg.Provider, llmCfg.Model)

	var result CallResult
	var callErr error
	if llmCfg.Provider == "claude" {
		result, callErr = CallClaude(client, llmCfg, systemPrompt, userPrompt)
	} else {
		result, callErr = CallOpenAICompat(client, llmCfg, systemPrompt, userPrompt)
	}
	if result.Error != "" {
		logging.Error("[LLM] 调用失败: %s", result.Error)
	} else {
		logging.Info("[LLM] 调用成功，响应长度=%d", len(result.Content))
	}
	return result, callErr
}

// CallVision 调用支持视觉的 LLM，可附带 base64 PNG 截图。
func CallVision(configID string, configs []config.LLMConfig, apiBaseURL, apiToken, systemPrompt, userPrompt, imageBase64 string) (CallResult, error) {
	if strings.HasPrefix(configID, "builtin:") {
		modelName := strings.TrimPrefix(configID, "builtin:")
		cfg, err := resolveBuiltinConfig(modelName, apiBaseURL, apiToken)
		if err != nil {
			return CallResult{Error: err.Error()}, err
		}
		client := &http.Client{Timeout: 60 * time.Second}
		logging.Info("[LLMVision] 调用内置模型 model=%s imageBase64长度=%d", modelName, len(imageBase64))
		var result CallResult
		var callErr error
		if imageBase64 == "" {
			result, callErr = CallOpenAICompat(client, cfg, systemPrompt, userPrompt)
		} else {
			result, callErr = CallOpenAICompatVision(client, cfg, systemPrompt, userPrompt, imageBase64)
		}
		if result.Error != "" {
			logging.Error("[LLMVision] 内置模型调用失败: %s", result.Error)
		} else {
			logging.Info("[LLMVision] 内置模型调用成功，响应长度=%d", len(result.Content))
		}
		return result, callErr
	}

	llmCfg := resolveConfig(configID, configs)
	if llmCfg == nil {
		return CallResult{Error: "no LLM config found"}, fmt.Errorf("no LLM config found")
	}

	client := &http.Client{Timeout: 60 * time.Second}
	logging.Info("[LLMVision] 调用模型 provider=%s model=%s imageBase64长度=%d", llmCfg.Provider, llmCfg.Model, len(imageBase64))

	var result CallResult
	var callErr error
	if imageBase64 == "" {
		if llmCfg.Provider == "claude" {
			result, callErr = CallClaude(client, llmCfg, systemPrompt, userPrompt)
		} else {
			result, callErr = CallOpenAICompat(client, llmCfg, systemPrompt, userPrompt)
		}
	} else if llmCfg.Provider == "claude" {
		result, callErr = CallClaudeVision(client, llmCfg, systemPrompt, userPrompt, imageBase64)
	} else {
		result, callErr = CallOpenAICompatVision(client, llmCfg, systemPrompt, userPrompt, imageBase64)
	}
	if result.Error != "" {
		logging.Error("[LLMVision] 调用失败: %s", result.Error)
	} else {
		logging.Info("[LLMVision] 调用成功，响应长度=%d", len(result.Content))
	}
	return result, callErr
}

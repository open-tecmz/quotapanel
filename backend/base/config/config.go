package config

import "quotapanel/backend/base/process"

// LLMConfig holds configuration for a single LLM provider/model.
type LLMConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"` // openai | gemini | claude
	APIKey    string `json:"apiKey"`
	BaseURL   string `json:"baseURL"`
	Model     string `json:"model"`
	IsDefault bool   `json:"isDefault"`
}

// BuiltinModelSetting 存储单个内置模型的用户偏好设置。
type BuiltinModelSetting struct {
	Name      string `json:"name"`
	Visible   bool   `json:"visible"`
	IsDefault bool   `json:"isDefault"`
}

type AppConfig struct {
	Locale               string                `json:"locale"`
	AutoStart            bool                  `json:"autoStart"`
	LogDir               string                `json:"logDir"`
	MaxLogLines          int                   `json:"maxLogLines"`
	MaxLogFiles          int                   `json:"maxLogFiles"`
	MaxRestart           int                   `json:"maxRestart"`
	RestartPolicy        string                `json:"restartPolicy"`
	DeviceUUID           string                `json:"deviceUUID"`
	SkillDir             string                `json:"skillDir"`
	AutoSyncToolIDs      []string              `json:"autoSyncToolIDs"`
	Processes            []process.Definition  `json:"processes"`
	LLMConfigs           []LLMConfig           `json:"llmConfigs"`
	BuiltinModelSettings []BuiltinModelSetting `json:"builtinModelSettings"`
	ServerPort           int                   `json:"serverPort"`
	ApiToken             string                `json:"apiToken"`
	// CloseAction is the default window close button behavior:
	// "" ask every time | "quit" quit directly | "hide" hide to background
	CloseAction string `json:"closeAction"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		Locale:               "zh",
		AutoStart:            false,
		LogDir:               "logs",
		MaxLogLines:          1000,
		MaxLogFiles:          5,
		MaxRestart:           5,
		RestartPolicy:        "on_failure",
		SkillDir:             "",
		AutoSyncToolIDs:      []string{},
		LLMConfigs:           []LLMConfig{},
		BuiltinModelSettings: []BuiltinModelSetting{},
		ServerPort:           6666,
	}
}

package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// quotaCall 处理 quota.* 命名空间的所有 Call 请求。
// 由 app_call.go 的 Call 方法路由过来。
func (a *App) quotaCall(name string, decode func(interface{}) error) (interface{}, error) {
	switch name {

	case "quota.getProviders":
		return a.quotaSvc.ListProviders(), nil

	case "quota.getAccounts":
		return a.quotaSvc.ListAccounts()

	case "quota.addAccount":
		var p struct {
			Provider string `json:"provider"`
			Name     string `json:"name"`
			Key      string `json:"key"`
			Note     string `json:"note"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return a.quotaSvc.AddAccount(p.Provider, p.Name, p.Key, p.Note)

	case "quota.updateAccount":
		var p struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
			Key  string `json:"key"`
			Note string `json:"note"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return a.quotaSvc.UpdateAccount(p.ID, p.Name, p.Key, p.Note)

	case "quota.deleteAccount":
		var p struct {
			ID uint `json:"id"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return nil, a.quotaSvc.DeleteAccount(p.ID)

	case "quota.queryAccount":
		var p struct {
			ID uint `json:"id"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return a.quotaSvc.QueryAccount(p.ID)

	case "quota.queryAll":
		return a.quotaSvc.QueryAll()

	case "quota.beginLogin":
		var p struct {
			ID uint `json:"id"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return nil, a.quotaSvc.BeginLogin(p.ID)

	case "quota.completeLogin":
		var p struct {
			ID uint `json:"id"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return a.quotaSvc.CompleteLogin(p.ID)

	case "quota.saveScreenshot":
		var p struct {
			AccountID uint   `json:"accountId"`
			PNGBase64 string `json:"pngBase64"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		if len(p.PNGBase64) > 14<<20 {
			return nil, fmt.Errorf("截图过大")
		}
		data, err := base64.StdEncoding.DecodeString(p.PNGBase64)
		if err != nil {
			return nil, fmt.Errorf("截图数据无效: %w", err)
		}
		if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
			return nil, fmt.Errorf("仅支持 PNG 截图")
		}
		filename := fmt.Sprintf("quota-card-%d.png", p.AccountID)
		path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: "保存卡片截图", DefaultFilename: filename, Filters: []runtime.FileFilter{{DisplayName: "PNG 图片", Pattern: "*.png"}}})
		if err != nil {
			return nil, err
		}
		if path == "" {
			return false, nil
		}
		return true, os.WriteFile(path, data, 0644)

	default:
		return nil, fmt.Errorf("未知方法: %s", name)
	}
}

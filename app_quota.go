package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"quotapanel/backend/base/platform"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// decodePNGBase64 校验并解码前端 Canvas 导出的 PNG。
func decodePNGBase64(value string) ([]byte, error) {
	if len(value) > 14<<20 {
		return nil, fmt.Errorf("截图过大")
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("截图数据无效: %w", err)
	}
	if len(data) < 8 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		return nil, fmt.Errorf("仅支持 PNG 截图")
	}
	return data, nil
}

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

	case "quota.reorderAccounts":
		var p struct {
			IDs []uint `json:"ids"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		return nil, a.quotaSvc.ReorderAccounts(p.IDs)

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
		snap, err := a.quotaSvc.QueryAccount(p.ID)
		if err != nil {
			return nil, err
		}
		a.storeQuotaSnapshot(snap)
		return snap, nil

	case "quota.queryAll":
		snaps, err := a.quotaSvc.QueryAll()
		if err != nil {
			return nil, err
		}
		a.storeQuotaSnapshots(snaps)
		platform.ReloadMiniPanel(a.buildMiniPanelHTML())
		return snaps, nil

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
		snap, err := a.quotaSvc.CompleteLogin(p.ID)
		if err != nil {
			return nil, err
		}
		a.storeQuotaSnapshot(snap)
		return snap, nil

	case "quota.saveScreenshot":
		var p struct {
			AccountID uint   `json:"accountId"`
			PNGBase64 string `json:"pngBase64"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		data, err := decodePNGBase64(p.PNGBase64)
		if err != nil {
			return nil, err
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

	case "quota.saveImage":
		var p struct {
			Title       string `json:"title"`
			DefaultName string `json:"defaultName"`
			PNGBase64   string `json:"pngBase64"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		data, err := decodePNGBase64(p.PNGBase64)
		if err != nil {
			return nil, err
		}
		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = "保存图片"
		}
		filename := filepath.Base(strings.TrimSpace(p.DefaultName))
		if filename == "" || filename == "." || filename == string(filepath.Separator) {
			filename = "quotapanel.png"
		}
		if !strings.HasSuffix(strings.ToLower(filename), ".png") {
			filename += ".png"
		}
		path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{Title: title, DefaultFilename: filename, Filters: []runtime.FileFilter{{DisplayName: "PNG 图片", Pattern: "*.png"}}})
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

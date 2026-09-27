package main

import (
	"quotapanel/backend/base/platform"
	"quotapanel/backend/quota"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// statusBarLeftClick 处理菜单栏 / 托盘图标左键单击：
// macOS 弹出额度迷你面板（不打开主窗口），其他平台沿用「显示主窗口」。
func (a *App) statusBarLeftClick() {
	if !platform.MiniPanelSupported() {
		a.showWindow()
		return
	}
	accounts := a.miniPanelAccounts()
	snaps := a.quotaSnapshots()
	platform.ShowMiniPanel(
		quota.BuildMiniPanelHTML(accounts, snaps, a.config.Locale),
		quota.MiniPanelHeight(accounts, snaps),
		platform.MiniPanelHandlers{OnAction: a.onMiniPanelAction},
	)
	// 打开面板时后台刷新一次，保证数据不过期；刷新完成后自动回填面板。
	go a.refreshQuotas()
}

// onMiniPanelAction 处理迷你面板内的交互（刷新 / 打开界面 / 点击账号）。
func (a *App) onMiniPanelAction(action string, accountID int) {
	switch action {
	case "refresh":
		go a.refreshQuotas()
	case "open":
		a.showWindow()
		if accountID > 0 && a.ctx != nil {
			// 通知前端定位到该账号的详情弹窗。
			runtime.EventsEmit(a.ctx, "mini:openAccount", accountID)
		}
	}
}

// buildMiniPanelHTML 用当前缓存的快照生成迷你面板 HTML。
func (a *App) buildMiniPanelHTML() string {
	return quota.BuildMiniPanelHTML(a.miniPanelAccounts(), a.quotaSnapshots(), a.config.Locale)
}

// miniPanelAccounts 返回迷你面板展示用的账号列表（保证非 nil）。
func (a *App) miniPanelAccounts() []quota.AccountView {
	accounts, err := a.quotaSvc.ListAccounts()
	if err != nil || accounts == nil {
		return []quota.AccountView{}
	}
	return accounts
}

// refreshQuotas 重新查询全部账号额度，更新缓存、通知前端并刷新迷你面板。
// 同一时间只允许一个全量查询，避免与定时轮询叠加。
func (a *App) refreshQuotas() {
	if a.quotaSvc == nil || !a.quotaQueryMu.TryLock() {
		return
	}
	defer a.quotaQueryMu.Unlock()

	snaps, err := a.quotaSvc.QueryAll()
	if err != nil {
		return
	}
	a.storeQuotaSnapshots(snaps)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "quota:snapshots", snaps)
	}
	platform.ReloadMiniPanel(a.buildMiniPanelHTML())
}

// storeQuotaSnapshots 合并写入额度快照缓存。
func (a *App) storeQuotaSnapshots(snaps []*quota.Snapshot) {
	a.quotaSnapMu.Lock()
	defer a.quotaSnapMu.Unlock()
	if a.quotaSnaps == nil {
		a.quotaSnaps = make(map[uint]*quota.Snapshot)
	}
	for _, s := range snaps {
		if s != nil {
			a.quotaSnaps[s.AccountID] = s
		}
	}
}

// storeQuotaSnapshot 写入单个账号的额度快照缓存。
func (a *App) storeQuotaSnapshot(s *quota.Snapshot) {
	if s == nil {
		return
	}
	a.storeQuotaSnapshots([]*quota.Snapshot{s})
}

// quotaSnapshots 返回快照缓存的副本。
func (a *App) quotaSnapshots() map[uint]*quota.Snapshot {
	a.quotaSnapMu.Lock()
	defer a.quotaSnapMu.Unlock()
	out := make(map[uint]*quota.Snapshot, len(a.quotaSnaps))
	for id, s := range a.quotaSnaps {
		out[id] = s
	}
	return out
}

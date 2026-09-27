package main

import (
	"context"
	"time"
)

// pollQuota 每 5 分钟自动刷新全部账号额度：
// 结果写入缓存、通过 quota:snapshots 通知前端，并同步刷新迷你面板。
func (a *App) pollQuota(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refreshQuotas()
		}
	}
}

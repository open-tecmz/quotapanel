package main

import (
	"context"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) pollQuota(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if a.quotaSvc == nil {
				continue
			}
			snaps, err := a.quotaSvc.QueryAll()
			if err == nil {
				runtime.EventsEmit(ctx, "quota:snapshots", snaps)
			}
		}
	}
}

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"quotapanel/backend/base/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// updateEndpoint returns the update backend URL. QUOTAPANEL_UPDATE_ENDPOINT
// overrides the built-in address so the whole flow can be exercised against a
// local test server without rebuilding version metadata.
func updateEndpoint() string {
	if value := strings.TrimSpace(os.Getenv("QUOTAPANEL_UPDATE_ENDPOINT")); value != "" {
		return value
	}
	return appConfig.VersionCheckUrl
}

// ensureUpdateManager creates the update manager once, after startup has
// loaded the config (device UUID) and the Wails context exists.
func (a *App) ensureUpdateManager() *updater.Manager {
	a.updateOnce.Do(func() {
		userAgent := fmt.Sprintf("AppOpen/%s/%s Platform/%s/%s/%s/%s",
			appConfig.Name, appConfig.Version,
			getPlatform(), getPlatformArch(), getPlatformVersion(), a.getDeviceUUID(),
		)
		a.updateMgr = updater.New(updater.Options{
			Endpoint:       updateEndpoint(),
			Repo:           updater.DefaultRepo,
			CurrentVersion: appConfig.Version,
			Channel:        "latest",
			UUID:           a.getDeviceUUID(),
			DataDir:        a.dataDir,
			Platform:       updater.CurrentPlatform(),
			UserAgent:      userAgent,
			OnEvent: func(ev updater.Event) {
				// The UI is optional (CLI-less tests, startup before mount), so a
				// missing context only means nobody is listening yet.
				if a.ctx != nil {
					runtime.EventsEmit(a.ctx, ev.Type, ev.State)
				}
			},
		})
	})
	return a.updateMgr
}

// updateContext returns the Wails context when available. Update operations
// outlive the renderer but must be cancelled when the app quits.
func (a *App) updateContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// updateCall handles the update.* namespace routed from Call.
func (a *App) updateCall(name string, decode func(interface{}) error) (interface{}, error) {
	manager := a.ensureUpdateManager()

	switch name {

	case "update.getState":
		return manager.State(), nil

	case "update.check":
		var p struct {
			Channel string `json:"channel"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		if p.Channel != "" {
			manager.SetChannel(p.Channel)
		}
		state, err := manager.Check(a.updateContext())
		if err != nil {
			return state, err
		}
		return state, nil

	case "update.setChannel":
		var p struct {
			Channel string `json:"channel"`
		}
		if err := decode(&p); err != nil {
			return nil, err
		}
		manager.SetChannel(p.Channel)
		return manager.State(), nil

	case "update.download":
		// Run in the background: progress and the final state reach the UI
		// through update:state events, exactly like the reference implementation.
		go func() {
			if _, err := manager.Download(context.Background()); err != nil {
				a.logSystemError("update", "download failed: "+err.Error())
			}
		}()
		return nil, nil

	case "update.install":
		result, _, err := manager.Install(context.Background())
		if err != nil {
			a.logSystemError("update", "install failed: "+err.Error())
			return result, err
		}
		if result.RestartRequired {
			// The platform installers start a detached helper that relaunches
			// the new version once this process exits (macOS / Windows / Linux
			// AppImage); quitting here lets that helper take over.
			a.quitting.Store(true)
			go func() {
				time.Sleep(400 * time.Millisecond)
				if a.ctx != nil {
					runtime.Quit(a.ctx)
				}
			}()
		}
		return result, nil

	default:
		return nil, fmt.Errorf("未知方法: %s", name)
	}
}

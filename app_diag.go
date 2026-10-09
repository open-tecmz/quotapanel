package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"quotapanel/backend/base/logging"
)

const (
	// rendererBootTimeout：窗口创建后等待渲染进程首次心跳的最长时间，
	// 超时即判定为「页面未加载成功（疑似白屏 / 加载失败）」。
	rendererBootTimeout = 25 * time.Second
	// rendererPingTimeout：运行期判定渲染进程失联（界面卡死 / 白屏）的心跳超时。
	rendererPingTimeout = 90 * time.Second
	// rendererWatchInterval：看门狗检查间隔。
	rendererWatchInterval = 30 * time.Second
	// metricsInterval：运行时内存 / 协程快照采样间隔。
	metricsInterval = 5 * time.Minute
)

// diagCall 处理 diag.* 诊断路由：前端心跳与本地日志桥接。
// 返回值：(result, error, matched)，matched=false 表示不属于 diag 路由。
func (a *App) diagCall(name string, decode func(interface{}) error) (interface{}, error, bool) {
	switch name {
	case "diag.log":
		var p struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
			Data  string `json:"data"`
		}
		if err := decode(&p); err != nil {
			return nil, err, true
		}
		a.logRenderer(p.Level, p.Msg, p.Data)
		return nil, nil, true

	case "diag.ping":
		a.markRendererAlive()
		return nil, nil, true
	}
	return nil, nil, false
}

// logRenderer 将前端日志写入应用运行日志（与后端日志同一文件），
// 未知级别按 info 处理；data 为可选的前端附带信息（如 stack）。
func (a *App) logRenderer(level, msg, data string) {
	if data != "" {
		msg = msg + " " + data
	}
	switch strings.ToLower(level) {
	case "error":
		logging.Error("[renderer] %s", msg)
	case "warn":
		logging.Warn("[renderer] %s", msg)
	case "debug":
		logging.Debug("[renderer] %s", msg)
	default:
		logging.Info("[renderer] %s", msg)
	}
}

// markRendererAlive 记录一次渲染进程心跳；首次心跳表示页面已成功加载并启动。
func (a *App) markRendererAlive() {
	first := a.rendererBooted.CompareAndSwap(false, true)
	a.rendererLastPing.Store(time.Now().UnixNano())
	if first {
		logging.Info("渲染进程已启动，首次心跳正常")
	}
}

// watchRenderer 渲染进程看门狗：
//   - 窗口创建后始终没有心跳 → 疑似白屏 / 页面加载失败；
//   - 运行期心跳长时间中断 → 界面疑似卡死。
//
// 两种情况都会写入日志并上报远程，便于排查偶发问题。
func (a *App) watchRenderer(ctx context.Context) {
	defer a.guardPanic("watchRenderer")

	deadline := time.Now().Add(rendererBootTimeout)
	for time.Now().Before(deadline) && !a.rendererBooted.Load() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}

	if !a.rendererBooted.Load() {
		msg := fmt.Sprintf("渲染进程在 %s 内未上报启动心跳，疑似白屏 / 页面加载失败", rendererBootTimeout)
		logging.Error("%s | %s", msg, runtimeSnapshot())
		a.ReportError(msg, "", "diagnostics/renderer-not-booted")
	} else {
		logging.Info("渲染进程启动正常")
	}

	ticker := time.NewTicker(rendererWatchInterval)
	defer ticker.Stop()
	reported := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		if a.quitting.Load() {
			continue
		}
		last := a.rendererLastPing.Load()
		if last == 0 {
			continue
		}
		silent := time.Since(time.Unix(0, last))
		if silent > rendererPingTimeout {
			if !reported {
				msg := fmt.Sprintf("渲染进程心跳中断 %s，界面疑似卡死 / 白屏", silent.Round(time.Second))
				logging.Error("%s | %s", msg, runtimeSnapshot())
				a.ReportError(msg, "", "diagnostics/renderer-stalled")
				reported = true
			}
		} else {
			reported = false
		}
	}
}

// sampleMetrics 定期记录 Go 运行时快照，便于发现长时间运行后的内存增长 /
// 协程泄漏（对应 Electron 侧的定时内存快照）。
func (a *App) sampleMetrics(ctx context.Context) {
	defer a.guardPanic("sampleMetrics")
	ticker := time.NewTicker(metricsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logging.Info("Diagnostics.metrics %s", runtimeSnapshot())
		}
	}
}

// runtimeSnapshot 返回当前 Go 运行时的内存 / 协程快照字符串。
func runtimeSnapshot() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return fmt.Sprintf(
		"goroutines=%d heap=%dMB sys=%dMB numGC=%d",
		runtime.NumGoroutine(),
		m.HeapAlloc/1024/1024,
		m.Sys/1024/1024,
		m.NumGC,
	)
}

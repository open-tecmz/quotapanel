package main

import (
	"fmt"
	"runtime/debug"

	"quotapanel/backend/base/logging"
)

// recordPanic 记录一次被 recover 的 panic：把消息与完整调用栈写入应用日志，
// 并上报到远程错误接口（stack 不再为空），返回格式化后的消息。
func (a *App) recordPanic(src string, r interface{}) string {
	stack := string(debug.Stack())
	msg := fmt.Sprintf("%s panic: %v", src, r)
	logging.Error("%s\n%s", msg, stack)
	a.ReportError(msg, stack, src)
	return msg
}

// guardPanic 是延迟 recover 处理器：捕获 panic 并记录，防止单个回调或后台任务
// 的异常直接终止整个应用。调用方自行决定是否重新抛出或转为错误。
func (a *App) guardPanic(src string) {
	if r := recover(); r != nil {
		a.recordPanic(src, r)
	}
}

// runGuarded 在当前 goroutine 中执行 fn，并把 panic 收敛到崩溃日志，
// 供后台任务使用（如额度刷新），避免一处异常拖垮进程。
func (a *App) runGuarded(src string, fn func()) {
	defer a.guardPanic(src)
	fn()
}

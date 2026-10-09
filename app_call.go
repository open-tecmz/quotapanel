package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Call 是前端统一调用入口（Wails 绑定），按命名空间路由到各模块。
// 名称格式：`namespace.methodName`，与前端 CallMap key 一一对应。
// args 是 JSON 编码的参数对象，各模块内部解析为强类型结构体。
func (a *App) Call(name string, args json.RawMessage) (result interface{}, callErr error) {
	// panic 兜底：将 panic 转为错误并上报（含堆栈）
	defer func() {
		if r := recover(); r != nil {
			a.recordPanic("Call/"+name, r)
			callErr = fmt.Errorf("内部错误: %v", r)
		}
	}()

	decode := func(v interface{}) error {
		if len(args) == 0 {
			return nil
		}
		return json.Unmarshal(args, v)
	}

	// 通用路由：llm.* / setting.*
	if result, err, ok := a.callCommon(name, decode); ok {
		return result, err
	}

	// diag.* 前端运行时诊断（心跳 / 本地日志桥接）
	if strings.HasPrefix(name, "diag.") {
		if result, err, ok := a.diagCall(name, decode); ok {
			return result, err
		}
		return nil, fmt.Errorf("未知方法: %s", name)
	}

	// quota.* 由 app_quota.go 处理
	if strings.HasPrefix(name, "quota.") {
		return a.quotaCall(name, decode)
	}

	return nil, fmt.Errorf("未知方法: %s", name)
}

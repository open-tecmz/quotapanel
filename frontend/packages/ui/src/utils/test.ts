/**
 * window.__test — UI 自动化测试辅助工具
 *
 * 核心思路：action 注册时机即页面就绪时机。
 * 测试侧通过 callAction 等待 action 被注册，无需额外的 ready 状态。
 *
 * 用法（页面组件 setup 中）：
 *   import { testActionSet, testActionUnset } from '@quotapanel/ui/src/utils/test'
 *   import { onMounted, onUnmounted } from 'vue'
 *
 *   onMounted(() => {
 *     testActionSet('QuotaAccount.getAccountCount', () => accounts.value.length)
 *   })
 *   onUnmounted(() => {
 *     testActionUnset('QuotaAccount.getAccountCount')
 *   })
 */

export type TestAction = (arg?: unknown) => Promise<unknown> | unknown

export interface TestRegistry {
  setAction(name: string, fn: TestAction): void
  unsetAction(name: string | string[]): void
  callAction(name: string, arg?: unknown): Promise<unknown>
  listActions(): string[]
  navigateTo(path: string): Promise<void>
}

const _actions = new Map<string, TestAction>()
let _navigateFn: ((path: string) => Promise<void>) | null = null

// ── console 日志拦截（仅测试模式激活后生效）──────────────────────────────────

export interface ConsoleLogEntry {
  level: 'error' | 'warn'
  message: string
  timestamp: number
}

const _consoleLogs: ConsoleLogEntry[] = []
let _testModeActive = false

/**
 * 激活测试模式：拦截 console.error/warn、全局错误事件。
 * 仅在测试程序调用 App.startTestMode 后执行一次，正常使用时不触发。
 */
function _activateTestMode() {
  if (_testModeActive) return
  _testModeActive = true

  const _origError = console.error.bind(console)
  const _origWarn = console.warn.bind(console)

  console.error = (...args: unknown[]) => {
    _origError(...args)
    _consoleLogs.push({
      level: 'error',
      message: args.map((a) => (typeof a === 'object' ? JSON.stringify(a) : String(a))).join(' '),
      timestamp: Date.now(),
    })
  }

  console.warn = (...args: unknown[]) => {
    _origWarn(...args)
    _consoleLogs.push({
      level: 'warn',
      message: args.map((a) => (typeof a === 'object' ? JSON.stringify(a) : String(a))).join(' '),
      timestamp: Date.now(),
    })
  }

  // 捕获全局未处理的 JS 错误
  window.addEventListener('error', (e) => {
    _consoleLogs.push({
      level: 'error',
      message: `[uncaught] ${e.message}${e.filename ? ` (${e.filename}:${e.lineno})` : ''}`,
      timestamp: Date.now(),
    })
  })

  // 捕获全局未处理的 Promise rejection
  window.addEventListener('unhandledrejection', (e) => {
    const reason = e.reason instanceof Error ? e.reason.message : String(e.reason)
    _consoleLogs.push({
      level: 'error',
      message: `[unhandledrejection] ${reason}`,
      timestamp: Date.now(),
    })
  })
}

/**
 * 主动上报一条错误日志（仅测试模式激活后有效）。
 */
export function reportTestError(msg: string): void {
  if (!_testModeActive) return
  _consoleLogs.push({ level: 'error', message: msg, timestamp: Date.now() })
}

/**
 * 主动上报一条警告日志（仅测试模式激活后有效）。
 */
export function reportTestWarn(msg: string): void {
  if (!_testModeActive) return
  _consoleLogs.push({ level: 'warn', message: msg, timestamp: Date.now() })
}

export const testRegistry: TestRegistry = {
  setAction(name, fn) {
    _actions.set(name, fn)
  },
  unsetAction(nameOrNames) {
    const names = Array.isArray(nameOrNames) ? nameOrNames : [nameOrNames]
    names.forEach((n) => _actions.delete(n))
  },
  async callAction(name, arg) {
    const fn = _actions.get(name)
    if (!fn) throw new Error(`[__test] action "${name}" 未注册`)
    return fn(arg)
  },
  listActions() {
    return Array.from(_actions.keys())
  },
  async navigateTo(path: string) {
    if (_navigateFn) {
      await _navigateFn(path)
    } else {
      window.location.hash = path
    }
  },
}

/**
 * 设置一个 test action。在 onMounted 中调用。
 * action 设置时机即页面就绪时机——测试侧 callAction 会等待 action 出现。
 */
export function testActionSet(name: string, fn: TestAction): void {
  testRegistry.setAction(name, fn)
}

/**
 * 移除一个或多个 test action。在 onUnmounted 中调用。
 */
export function testActionUnset(nameOrNames: string | string[]): void {
  testRegistry.unsetAction(nameOrNames)
}

/**
 * 注册 Vue Router 导航函数，在 App.vue 的 onMounted 中调用。
 */
export function registerNavigate(fn: (path: string) => Promise<void>): void {
  _navigateFn = fn
}

/**
 * 挂载 window.__test，在 App.vue 的 onMounted 中调用。
 * 注册 App.getConsoleLogs / App.clearConsoleLogs / App.startTestMode action。
 * 拦截逻辑不在此处激活，需测试程序调用 App.startTestMode 后才生效。
 */
export function initTestRegistry(opts: { onStartTestMode: () => void }): void {
  ;(window as unknown as Record<string, unknown>).__test = testRegistry
  testRegistry.setAction('App.getConsoleLogs', () => [..._consoleLogs])
  testRegistry.setAction('App.clearConsoleLogs', () => {
    _consoleLogs.length = 0
  })
  testRegistry.setAction('App.startTestMode', () => {
    _activateTestMode()
    opts.onStartTestMode()
  })
}

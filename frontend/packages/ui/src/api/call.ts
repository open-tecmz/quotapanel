/**
 * 统一前后端调用入口
 *
 * 用法：
 *   import { main } from '@quotapanel/ui/src/api/call'
 *   const info = await main.Call('setting.getAppConfig')
 *   await main.Call('setting.setLocale', { locale: 'zh' })
 */

// ─── 数据类型 ───

export interface UserInfo {
  id: number
  avatar: string
  viewName: string
}

export interface VersionInfo {
  name: string
  version: string
  time: string
  url?: string
}

export interface BrowserInfo {
  id: string
  name: string
  path: string
  found: boolean
}

export interface SessionInfo {
  id: string
  name: string
  browserPath: string
  debugPort: number
  connected: boolean
}

export interface TabInfo {
  targetId: string
  title: string
  url: string
  type: string
}

export interface EvalResult {
  value: string
  error?: string
}

export interface PickResult {
  selector: string
  tagName: string
  innerText: string
  outerHTML: string
  confirmed: boolean
}

export interface Task {
  id: number
  name: string
  script: string
  params: string // JSON 序列化的参数定义列表
  triggerMode: string
  cronExpression: string
  enabled: boolean
  status: string
  createdAt: string
  updatedAt: string
}

export interface TaskLog {
  id: number
  taskId: number
  status: string
  output: string
  durationMs: number
  startedAt: string
  finishedAt: string
}

export interface TaskLogPage {
  items: TaskLog[]
  total: number
  page: number
  size: number
}

export interface LLMCallResult {
  content: string
  error?: string
}

export interface LLMConfig {
  id: string
  name: string
  provider: string
  apiKey: string
  baseURL: string
  model: string
  isDefault: boolean
}

export interface BuiltinModelSetting {
  name: string
  visible: boolean
  isDefault: boolean
}

export interface LLMPXModelInfo {
  id: string
  name: string
  rate: number
}

// ─── 额度（quota）───

export interface QuotaProviderInfo {
  id: string
  name: string
  description: string
  mode: 'api' | 'browser'
  keyLabel: string
  keyHint: string
  docsUrl: string
}

export interface QuotaAccount {
  id: number
  provider: string
  providerName: string
  name: string
  note: string
  key: string
  keyMasked: string
  createdAt: string
  updatedAt: string
}

export interface QuotaWindow {
  key: string
  label: string
  percent: number
  status: 'ok' | 'warning' | 'exceeded' | string
  detail: string
  resetAt: string
}

export interface QuotaStat {
  label: string
  value: string
  muted: boolean
}

export interface QuotaSnapshot {
  accountId: number
  provider: string
  accountName: string
  plan: string
  status: string
  windows: QuotaWindow[]
  balances: QuotaStat[]
  stats: QuotaStat[]
  summary: string
  updatedAt: string
  error?: string
}

export interface VipInfo {
  id: string | null
  flag: string | null
  title: string | null
  isDefault: boolean
  icon: string | null
}

export interface LLMPXInfo {
  quota: number
  apiUrl: string
  apiKey: string
  models: LLMPXModelInfo[]
  vip: VipInfo | null
}

// ─── 调用签名映射 ───

export interface CallMap {
  // ── browser：浏览器管理 ──────────────────────────────────
  'browser.scanBrowsers': { args: void; ret: BrowserInfo[] }
  'browser.launchBrowser': {
    args: { path: string; port: number; name?: string }
    ret: string
  }
  'browser.connectBrowser': {
    args: { port: number; name?: string }
    ret: string
  }
  'browser.disconnectBrowser': { args: { sessionID: string }; ret: void }
  'browser.getBrowserSessions': { args: void; ret: SessionInfo[] }
  'browser.renameSession': {
    args: { sessionID: string; name: string }
    ret: void
  }
  'browser.ensureSessionByName': { args: { name: string }; ret: string }

  // ── browser：Tab 管理 ────────────────────────────────────
  'browser.getTabs': { args: { sessionID: string }; ret: TabInfo[] }
  'browser.newTab': { args: { sessionID: string; url: string }; ret: TabInfo }
  'browser.closeTab': {
    args: { sessionID: string; targetID: string }
    ret: void
  }
  'browser.activateTab': {
    args: { sessionID: string; targetID: string }
    ret: void
  }

  // ── browser：页面操作 ────────────────────────────────────
  'browser.navigate': {
    args: { sessionID: string; targetID: string; url: string }
    ret: void
  }
  'browser.evalJS': {
    args: { sessionID: string; targetID: string; code: string }
    ret: EvalResult
  }
  'browser.clickElement': {
    args: { sessionID: string; targetID: string; selector: string }
    ret: void
  }
  'browser.setText': {
    args: {
      sessionID: string
      targetID: string
      selector: string
      text: string
    }
    ret: void
  }
  'browser.keyPress': {
    args: { sessionID: string; targetID: string; key: string }
    ret: void
  }
  'browser.waitForElement': {
    args: {
      sessionID: string
      targetID: string
      selector: string
      timeoutMs: number
    }
    ret: void
  }
  'browser.captureScreenshot': {
    args: { sessionID: string; targetID: string }
    ret: string
  }

  // ── browser：元素拾取 ────────────────────────────────────
  'browser.startPickElement': {
    args: { sessionID: string; targetID: string }
    ret: void
  }
  'browser.pollPickedElement': {
    args: { sessionID: string; targetID: string }
    ret: PickResult
  }
  'browser.stopPickElement': {
    args: { sessionID: string; targetID: string }
    ret: void
  }

  // ── task：任务管理 ───────────────────────────────────────
  'task.createTask': { args: { task: Partial<Task> }; ret: Task }
  'task.getTaskList': { args: void; ret: Task[] }
  'task.getTask': { args: { id: number }; ret: Task }
  'task.updateTask': { args: { task: Task }; ret: Task }
  'task.deleteTask': { args: { id: number }; ret: void }
  'task.runTask': {
    args: { id: number; params?: Record<string, string> }
    ret: void
  }
  'task.getTaskLogs': {
    args: {
      taskID: number
      page: number
      size: number
      status?: string
      dateStart?: string
      dateEnd?: string
    }
    ret: TaskLogPage
  }

  // ── service：HTTP 服务 ───────────────────────────────────
  'service.getServerPort': { args: void; ret: number }
  'service.setServerPort': { args: { port: number }; ret: void }
  'service.isHttpServerRunning': { args: void; ret: boolean }
  'service.restartHttpServer': { args: void; ret: void }
  'service.stopHttpServer': { args: void; ret: void }
  'service.getApiDoc': { args: void; ret: string }

  // ── llm：大模型 ──────────────────────────────────────────
  'llm.getLLMConfigs': { args: void; ret: LLMConfig[] }
  'llm.saveLLMConfigs': { args: { configs: LLMConfig[] }; ret: void }
  'llm.testLLMConfig': {
    args: { provider: string; apiKey: string; baseURL: string; model: string }
    ret: LLMCallResult
  }
  'llm.callLLM': {
    args: { configID: string; systemPrompt: string; userPrompt: string }
    ret: LLMCallResult
  }
  'llm.callLLMVision': {
    args: {
      configID: string
      systemPrompt: string
      userPrompt: string
      imageBase64: string
    }
    ret: LLMCallResult
  }
  'llm.getLLMPXInfo': { args: void; ret: LLMPXInfo }
  'llm.getBuiltinModelSettings': { args: void; ret: BuiltinModelSetting[] }
  'llm.saveBuiltinModelSettings': {
    args: { settings: BuiltinModelSetting[] }
    ret: void
  }

  // ── setting：应用设置 ────────────────────────────────────
  'setting.getAppConfig': { args: void; ret: Record<string, any> }
  'setting.checkVersion': { args: void; ret: VersionInfo }
  'setting.getAutoStartEnabled': { args: void; ret: boolean }
  'setting.setAutoStartEnabled': { args: { enabled: boolean }; ret: void }
  'setting.getApiToken': { args: void; ret: string }
  'setting.setApiToken': { args: { token: string }; ret: void }
  'setting.getSystemVersion': { args: void; ret: Record<string, string> }
  'setting.getUserInfo': { args: void; ret: UserInfo }
  'setting.sendAnalytics': {
    args: { events: Array<{ name: string; data?: Record<string, any> }> }
    ret: void
  }
  'setting.getLocale': { args: void; ret: string }
  'setting.setLocale': { args: { locale: string }; ret: void }
  'setting.showWindow': { args: void; ret: void }
  'setting.hideWindow': { args: void; ret: void }
  'setting.quitApp': { args: void; ret: void }
  'setting.requestClose': { args: void; ret: void }
  'setting.restartApp': { args: void; ret: void }
  'setting.getCloseAction': { args: void; ret: string }
  'setting.setCloseAction': { args: { action: string }; ret: void }
  'setting.resolveClose': {
    args: { action: 'quit' | 'hide'; remember: boolean }
    ret: void
  }
  'setting.selectDirectory': { args: void; ret: string }
  'setting.selectFile': { args: void; ret: string }
  'setting.selectZipFile': { args: void; ret: string }
  'setting.saveLogsToFile': {
    args: { processName: string; content: string }
    ret: void
  }
  'setting.getSystemLogs': { args: void; ret: string }
  'setting.getPlatformName': {
    args: void
    ret: 'win' | 'osx' | 'linux' | 'none'
  }
  'setting.getPlatformArch': { args: void; ret: 'x86' | 'arm64' | 'none' }

  // ── quota：AI 订阅额度 ────────────────────────────────────
  'quota.getProviders': { args: void; ret: QuotaProviderInfo[] }
  'quota.getAccounts': { args: void; ret: QuotaAccount[] }
  'quota.addAccount': {
    args: { provider: string; name: string; key: string; note: string }
    ret: QuotaAccount
  }
  'quota.updateAccount': {
    args: { id: number; name: string; key: string; note: string }
    ret: QuotaAccount
  }
  'quota.deleteAccount': { args: { id: number }; ret: void }
  'quota.queryAccount': { args: { id: number }; ret: QuotaSnapshot }
  'quota.queryAll': { args: void; ret: QuotaSnapshot[] }
  'quota.beginLogin': { args: { id: number }; ret: void }
  'quota.completeLogin': { args: { id: number }; ret: QuotaSnapshot }
  'quota.saveScreenshot': { args: { accountId: number; pngBase64: string }; ret: boolean }
}

// ─── 统一调用入口 ───

export const main = {
  Call<K extends keyof CallMap>(
    name: K,
    ...rest: CallMap[K]['args'] extends void ? [] : [args: CallMap[K]['args']]
  ): Promise<CallMap[K]['ret']> {
    const goCall = (window as any)?.go?.main?.App?.Call
    if (!goCall) {
      return Promise.reject(new Error('Call 方法未找到，请确认 Wails 已正确初始化'))
    }
    const args = rest.length > 0 ? rest[0] : undefined
    return goCall(name, args)
  },
}

/**
 * 错误上报工具
 *
 * 通过 HTTP Beacon 接口批量上报前端错误，延迟 5s 合并发送，避免频繁请求。
 * type 格式：app-<version>，版本号从后端动态获取并缓存。
 */

const BEACON_URL = 'https://g.tecmz.com/grow/load.gif'
const APP_ID = 'quotapanel'
const FLUSH_DELAY = 5000

// ─── 设备 / 会话 ID ────────────────────────────────────────────────────────

function getDeviceId(): string {
  let did = localStorage.getItem('__err_did')
  if (!did) {
    did = 'did_' + Math.random().toString(36).slice(2) + Date.now().toString(36)
    localStorage.setItem('__err_did', did)
  }
  return did
}

const SESSION_ID = 'sid_' + Math.random().toString(36).slice(2) + Date.now().toString(36)

// ─── 版本号缓存 ────────────────────────────────────────────────────────────

let _appVersion: string | null = null

async function getAppVersion(): Promise<string> {
  if (_appVersion) return _appVersion
  try {
    const goCall = (window as any)?.go?.main?.App?.Call
    if (goCall) {
      const cfg = await goCall('setting.getAppConfig', undefined)
      _appVersion = (cfg?.version as string) || 'unknown'
    } else {
      _appVersion = 'unknown'
    }
  } catch {
    _appVersion = 'unknown'
  }
  return _appVersion!
}

// ─── 事件结构 ──────────────────────────────────────────────────────────────

export interface ErrorEvent {
  et: 'error'
  path: string
  did: string
  sid: string
  ts: number
  type: string
  props: {
    msg: string
    stack?: string
    src?: string
    line?: number
    col?: number
    build_id?: string
  }
}

// ─── 批量队列 ──────────────────────────────────────────────────────────────

const _queue: ErrorEvent[] = []
let _timer: ReturnType<typeof setTimeout> | null = null

function scheduleFlush(): void {
  if (_timer) return
  _timer = setTimeout(() => {
    _timer = null
    flush()
  }, FLUSH_DELAY)
}

async function flush(): Promise<void> {
  if (_queue.length === 0) return
  const batch = _queue.splice(0, _queue.length)
  try {
    const payload = JSON.stringify(batch)
    const b64 = btoa(unescape(encodeURIComponent(payload)))
    const encoded = encodeURIComponent(b64)
    const url = `${BEACON_URL}?app=${APP_ID}&data=${encoded}`
    // 优先用 sendBeacon，失败时降级 fetch（fire-and-forget）
    if (navigator.sendBeacon) {
      navigator.sendBeacon(url)
    } else {
      fetch(url, { method: 'GET', keepalive: true }).catch(() => undefined)
    }
  } catch {
    // 上报失败静默处理，避免二次报错
  }
}

// 页面卸载前强制上报剩余事件
window.addEventListener('beforeunload', () => {
  if (_queue.length > 0) flush()
})

// ─── 公开接口 ──────────────────────────────────────────────────────────────

export async function reportError(options: {
  msg: string
  stack?: string
  src?: string
  line?: number
  col?: number
  buildId?: string
}): Promise<void> {
  const version = await getAppVersion()
  const event: ErrorEvent = {
    et: 'error',
    path: window.location.hash.replace(/^#/, '') || '/',
    did: getDeviceId(),
    sid: SESSION_ID,
    ts: Date.now(),
    type: `app-${version}`,
    props: {
      msg: options.msg,
      stack: options.stack,
      src: options.src,
      line: options.line,
      col: options.col,
      build_id: options.buildId || (import.meta.env.VITE_BUILD_ID as string | undefined),
    },
  }
  // 去掉 undefined 字段
  Object.keys(event.props).forEach((k) => {
    const key = k as keyof typeof event.props
    if (event.props[key] === undefined) delete event.props[key]
  })
  _queue.push(event)
  scheduleFlush()
}

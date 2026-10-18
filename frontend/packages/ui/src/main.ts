import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import { i18n } from './plugins/i18n'
import { router } from './router'
import './style.css'
import { trackError, trackOpen } from './utils/analytics'
import { reportError } from './utils/error-reporter'

// ── 运行时诊断：先写本地日志，再远程上报 ──────────────────────────────────
// boot 标记写在最前：若日志里没有它，说明页面根本没加载起来（资源 / 加载失败）；
// 有 boot 但没有后续 mount 相关标记，则说明崩在挂载阶段。
function backendDiag(level: 'info' | 'warn' | 'error' | 'debug', msg: string, data?: unknown) {
  try {
    const goCall = (window as any)?.go?.main?.App?.Call
    if (goCall) goCall('diag.log', { level, msg, data: data ? JSON.stringify(data) : '' })
  } catch {
    /* 诊断失败静默处理，避免二次报错 */
  }
}

// 渲染进程心跳：Go 侧看门狗据此判断界面是否白屏 / 卡死。
function pingRenderer() {
  try {
    const goCall = (window as any)?.go?.main?.App?.Call
    if (goCall) goCall('diag.ping', undefined)
  } catch {
    /* ignore */
  }
}

backendDiag('info', 'boot', { href: location.href, ua: navigator.userAgent })
pingRenderer()
setInterval(pingRenderer, 30000)

window.addEventListener('beforeunload', () => backendDiag('info', 'close'))

const app = createApp(App)
app.use(createPinia())
app.use(i18n)
app.use(Antd)
app.use(router)

// Track app open event
trackOpen()

// Global error handler for Vue
app.config.errorHandler = (err, _instance, info) => {
  const errorMessage = err instanceof Error ? err.message : String(err)
  const stack = err instanceof Error ? err.stack : undefined
  backendDiag('error', `Vue Error: ${errorMessage} (${info})`, { stack })
  trackError(`Vue Error: ${errorMessage} (${info})`)
  reportError({ msg: `Vue Error: ${errorMessage} (${info})`, stack })
  console.error('Vue Error:', err, info)
}

// Router navigation errors (bad route / lazy chunk load failure).
router.onError((err) => {
  const errorMessage = err instanceof Error ? err.message : String(err)
  const stack = err instanceof Error ? err.stack : undefined
  backendDiag('error', `Router Error: ${errorMessage}`, { stack })
  trackError(`Router Error: ${errorMessage}`)
  reportError({ msg: `Router Error: ${errorMessage}`, stack })
  console.error('Router Error:', err)
})

// Global unhandled promise rejection handler
window.addEventListener('unhandledrejection', (event) => {
  const reason = event.reason
  const errorMessage = reason instanceof Error ? reason.message : String(reason)
  const stack = reason instanceof Error ? reason.stack : undefined
  backendDiag('error', `Unhandled Promise Rejection: ${errorMessage}`, { stack })
  trackError(`Unhandled Promise Rejection: ${errorMessage}`)
  reportError({ msg: `Unhandled Promise Rejection: ${errorMessage}`, stack })
})

// Global error handler for uncaught errors
window.addEventListener('error', (event) => {
  const stack = event.error instanceof Error ? event.error.stack : undefined
  backendDiag('error', `Uncaught Error: ${event.message}`, {
    stack,
    src: event.filename,
    line: event.lineno,
    col: event.colno,
  })
  trackError(`Uncaught Error: ${event.message}`)
  reportError({
    msg: `Uncaught Error: ${event.message}`,
    src: event.filename,
    line: event.lineno,
    col: event.colno,
    stack,
  })
})

app.mount('#app')
backendDiag('info', 'mounted')

// 挂载完成后淡出并移除启动加载屏
const loadingEl = document.getElementById('app-loading')
if (loadingEl) {
  loadingEl.style.opacity = '0'
  setTimeout(() => loadingEl.remove(), 260)
}

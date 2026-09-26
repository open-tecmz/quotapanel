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
  trackError(`Vue Error: ${errorMessage} (${info})`)
  reportError({ msg: `Vue Error: ${errorMessage} (${info})`, stack })
  console.error('Vue Error:', err, info)
}

// Global unhandled promise rejection handler
window.addEventListener('unhandledrejection', (event) => {
  const reason = event.reason
  const errorMessage = reason instanceof Error ? reason.message : String(reason)
  const stack = reason instanceof Error ? reason.stack : undefined
  trackError(`Unhandled Promise Rejection: ${errorMessage}`)
  reportError({ msg: `Unhandled Promise Rejection: ${errorMessage}`, stack })
})

// Global error handler for uncaught errors
window.addEventListener('error', (event) => {
  trackError(`Uncaught Error: ${event.message}`)
  reportError({
    msg: `Uncaught Error: ${event.message}`,
    src: event.filename,
    line: event.lineno,
    col: event.colno,
    stack: event.error instanceof Error ? event.error.stack : undefined,
  })
})

app.mount('#app')

// 挂载完成后淡出并移除启动加载屏
const loadingEl = document.getElementById('app-loading')
if (loadingEl) {
  loadingEl.style.opacity = '0'
  setTimeout(() => loadingEl.remove(), 260)
}

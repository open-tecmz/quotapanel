<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { main } from '../api/call'

// 工单反馈弹窗：内嵌官网 feedback_ticket 页面（iframe），
// 并通过 postMessage 响应页面请求，回传运行环境与应用日志。
defineProps<{ open: boolean; url: string }>()

const emit = defineEmits<{ 'update:open': [boolean] }>()

const { t } = useI18n()

function reply(source: MessageEventSource | null, type: string, data: unknown) {
  if (source && 'postMessage' in source) {
    ;(source as Window).postMessage({ type, data }, '*')
  }
}

async function sendEnv(source: MessageEventSource | null) {
  try {
    const cfg = (await main.Call('setting.getAppConfig')) || {}
    reply(source, 'FeedbackTicket:env', {
      name: String(cfg.title || cfg.name || 'QuotaPanel'),
      version: String(cfg.version || ''),
      platform: await main.Call('setting.getPlatformName'),
      systemVersion: await main.Call('setting.getSystemVersion'),
    })
  } catch (error) {
    console.error('收集反馈环境信息失败:', error)
  }
}

async function sendLog(source: MessageEventSource | null) {
  try {
    const logs = (await main.Call('setting.getSystemLogs')) || ''
    const now = new Date()
    reply(source, 'FeedbackTicket:log', {
      logs: String(logs),
      startTime: new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString(),
      endTime: now.toISOString(),
    })
  } catch (error) {
    console.error('收集反馈日志失败:', error)
  }
}

function onMessage(event: MessageEvent) {
  const type = (event.data as { type?: string } | null)?.type
  if (!type) return
  if (type === 'FeedbackTicket:env') sendEnv(event.source)
  else if (type === 'FeedbackTicket:log') sendLog(event.source)
}

onMounted(() => window.addEventListener('message', onMessage))
onUnmounted(() => window.removeEventListener('message', onMessage))
</script>

<template>
  <a-modal
    :open="open"
    :title="t('settings.feedback')"
    :footer="null"
    width="min(720px, 90vw)"
    :style="{ top: '20px' }"
    :body-style="{ padding: 0, height: '70vh', overflow: 'visible' }"
    @update:open="emit('update:open', $event)"
  >
    <div class="feedback-frame">
      <iframe v-if="url" :src="url" title="feedback"></iframe>
    </div>
  </a-modal>
</template>

<style scoped>
.feedback-frame {
  width: calc(48px + 100%);
  height: calc(20px + 100%);
  margin: 0 24px -20px -24px;
  overflow: hidden;
  border-radius: 0 0 8px 8px;
}
.feedback-frame iframe {
  width: 100%;
  height: 100%;
  border: 0;
}
</style>

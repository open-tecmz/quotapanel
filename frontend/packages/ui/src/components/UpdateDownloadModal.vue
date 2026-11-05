<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  bindUpdateEvents,
  hideUpdateToBackground,
  installUpdate,
  reopenUpdateModal,
  updateStore,
  updateUI,
} from '../utils/version'

const { t } = useI18n()
const state = computed(() => updateStore.state)
const installError = ref('')

const visible = computed({
  get: () => updateUI.modalVisible,
  set: (value: boolean) => {
    updateUI.modalVisible = value
  },
})

const downloading = computed(() => state.value?.state === 'downloading')
const ready = computed(() => state.value?.state === 'ready')
const installing = computed(() => state.value?.state === 'installing' || state.value?.state === 'restarting')
const failed = computed(() => state.value?.state === 'failed' && !!state.value?.error)
const dependencyError = computed(() => (state.value?.error || '').includes('missing system dependency'))
const errorDescription = computed(() => (state.value?.error || '').replace(/^install:\s*/, ''))

const progressStatus = computed<'active' | 'success' | 'normal'>(() => {
  if (downloading.value) return 'active'
  if (ready.value) return 'success'
  return 'normal'
})

const hintText = computed(() => {
  if (installing.value) return t('settings.version.installing')
  if (ready.value) return t('settings.version.updateReady')
  return t('settings.version.downloading')
})

const floatText = computed(() => {
  if (downloading.value) {
    return t('settings.version.backgroundDownloading', {
      progress: state.value?.progress || 0,
    })
  }
  return t('settings.version.updateReadyShort')
})

function formatBytes(value: number): string {
  if (!value || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

function handleBackground() {
  hideUpdateToBackground()
}

function handleLater() {
  updateUI.modalVisible = false
  updateUI.minimized = true
}

function handleClose() {
  updateUI.modalVisible = false
  updateUI.minimized = false
}

async function handleInstall() {
  installError.value = ''
  try {
    await installUpdate()
  } catch (e) {
    // The backend keeps the "ready" state so the install can be retried.
    installError.value = e instanceof Error ? e.message : String(e)
  }
}

onMounted(() => {
  bindUpdateEvents()
})
</script>

<template>
  <a-modal
    :open="visible"
    :title="t('settings.version.updateDownloadTitle')"
    :closable="false"
    :mask-closable="false"
    :footer="null"
    width="min(520px, 92vw)"
    wrap-class-name="update-download-modal"
  >
    <div class="update-modal">
      <div class="version-row">
        <a-tag color="green">{{ state?.latestVersion }}</a-tag>
        <span class="summary" :title="state?.release?.summary">{{ state?.release?.summary }}</span>
      </div>

      <template v-if="failed">
        <a-alert
          type="error"
          show-icon
          :message="dependencyError ? t('settings.version.missingDependency') : t('settings.version.updateFailed')"
          :description="errorDescription"
        />
        <div class="actions">
          <a-button v-if="state?.downloadedPath" @click="handleInstall">
            {{ t('settings.version.retryInstall') }}
          </a-button>
          <a-button @click="handleClose">{{ t('settings.version.close') }}</a-button>
        </div>
      </template>

      <template v-else>
        <a-progress :percent="state?.progress || 0" :status="progressStatus" />
        <div v-if="downloading" class="meta">
          {{ formatBytes(state?.downloaded || 0) }} / {{ formatBytes(state?.total || 0) }}
        </div>
        <div class="hint">{{ hintText }}</div>

        <a-alert
          v-if="installError"
          type="error"
          show-icon
          :message="t('settings.version.installFailed')"
          :description="installError"
        />

        <div class="actions">
          <a-button v-if="downloading" @click="handleBackground">
            {{ t('settings.version.downloadInBackground') }}
          </a-button>
          <template v-if="ready">
            <a-button @click="handleLater">{{ t('settings.version.later') }}</a-button>
            <a-button type="primary" :loading="installing" @click="handleInstall">
              {{ t('settings.version.restartAndUpdate') }}
            </a-button>
          </template>
        </div>
      </template>
    </div>
  </a-modal>

  <div v-if="updateUI.minimized && (downloading || ready)" class="update-float" @click="reopenUpdateModal">
    <span>{{ floatText }}</span>
    <a-progress v-if="downloading" type="circle" :percent="state?.progress || 0" :size="26" :show-info="false" />
  </div>
</template>

<style scoped>
.update-modal {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.version-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.summary {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--token-neutral-500);
}

.meta {
  font-size: 12px;
  color: var(--token-neutral-500);
}

.hint {
  font-size: 13px;
  color: var(--token-neutral-700);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 4px;
}

.update-float {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 50;
  display: flex;
  cursor: pointer;
  align-items: center;
  gap: 8px;
  border-radius: 9999px;
  background: var(--token-primary-500);
  padding: 8px 16px;
  font-size: 12px;
  font-weight: 500;
  color: #fff;
  box-shadow: 0 8px 24px rgb(0 0 0 / 18%);
  transition: background 0.15s ease;
}

.update-float:hover {
  background: var(--token-primary-600);
}

:global(.dark .summary),
:global(.dark .meta) {
  color: var(--token-neutral-400);
}

:global(.dark .hint) {
  color: var(--token-neutral-300);
}
</style>

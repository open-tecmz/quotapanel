<template>
  <a-modal
    :open="open"
    :title="title"
    :footer="null"
    width="min(920px, 92vw)"
    wrap-class-name="share-preview-modal"
    @cancel="close"
  >
    <div class="preview-scroll">
      <img v-if="pngBase64" class="preview-image" :src="`data:image/png;base64,${pngBase64}`" :alt="title" />
    </div>
    <div class="preview-footer">
      <span class="preview-hint">{{ hint }}</span>
      <div class="preview-actions">
        <a-button @click="close">{{ t('quota.account.close') }}</a-button>
        <a-button type="primary" :loading="saving" @click="emit('save')">
          <span class="inline-flex items-center gap-1">
            <Download class="w-4 h-4" aria-hidden="true" />
            {{ t('quota.account.saveImage') }}
          </span>
        </a-button>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { Download } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  open: boolean
  pngBase64: string
  saving: boolean
  title: string
  hint: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  save: []
}>()

function close() {
  emit('update:open', false)
}
</script>

<style scoped>
.preview-scroll {
  max-height: 62vh;
  overflow-y: auto;
  padding: 4px 4px 12px;
  background: var(--token-neutral-100);
  border-radius: 10px;
}

.preview-image {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(var(--token-neutral-800-rgb), 0.16);
}

.preview-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 14px;
  flex-wrap: wrap;
}

.preview-hint {
  font-size: 12px;
  color: var(--token-neutral-400);
}

.preview-actions {
  display: flex;
  gap: 8px;
}

:global(.dark .preview-scroll) {
  background: var(--token-panel);
}

:global(.dark .preview-hint) {
  color: var(--token-neutral-400);
}
</style>

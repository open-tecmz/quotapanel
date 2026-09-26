<script setup lang="ts">
import { LogOut, Minimize2 } from 'lucide-vue-next'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { main } from '../api/call'

// Close confirmation dialog: ask whether to quit or hide to background,
// with a "remember" checkbox.
defineProps<{ open: boolean }>()

const emit = defineEmits<{ 'update:open': [boolean] }>()

const { t } = useI18n()
const remember = ref(false)

async function doChoose(action: 'quit' | 'hide') {
  emit('update:open', false)
  await main.Call('setting.resolveClose', { action, remember: remember.value })
}

function onOpenChange(value: boolean) {
  emit('update:open', value)
}
</script>

<template>
  <a-modal
    :open="open"
    :title="t('settings.closeConfirm.title')"
    :footer="null"
    width="min(600px, 90vw)"
    wrap-class-name="close-confirm-modal"
    @update:open="onOpenChange"
  >
    <div class="close-confirm-desc">{{ t('settings.closeConfirm.desc') }}</div>
    <div class="close-confirm-actions">
      <a-button @click="doChoose('hide')">
        <span class="btn-inner">
          <Minimize2 class="w-4 h-4" aria-hidden="true" />
          {{ t('settings.closeConfirm.hide') }}
        </span>
      </a-button>
      <a-button type="primary" danger @click="doChoose('quit')">
        <span class="btn-inner">
          <LogOut class="w-4 h-4" aria-hidden="true" />
          {{ t('settings.closeConfirm.quit') }}
        </span>
      </a-button>
    </div>
    <a-checkbox v-model:checked="remember" class="close-confirm-remember">
      {{ t('settings.closeConfirm.remember') }}
    </a-checkbox>
  </a-modal>
</template>

<style scoped>
.close-confirm-desc {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 16px;
}
.close-confirm-actions {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
.close-confirm-actions :deep(.ant-btn) {
  flex: 1;
}
.btn-inner {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.close-confirm-remember {
  font-size: 13px;
}
:global(.dark .close-confirm-desc) {
  color: #9ca3af;
}
</style>

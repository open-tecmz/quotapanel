<template>
  <div class="llm-config-panel">
    <div class="card">
      <div class="card-section-header">
        <div class="section-title-row">
          <BrainCircuit :size="16" aria-hidden="true" />
          <span>模型大配置</span>
        </div>
        <a-button @click="doAddLLM">
          <div class="inline-flex items-center gap-1">
            <Plus class="w-4 h-4" aria-hidden="true" />
            新增
          </div>
        </a-button>
      </div>

      <!-- 内置模型横条 -->

      <!-- 自定义模型区域 -->
      <div class="custom-section">
        <div class="custom-section-header">
          <span class="custom-section-label">自定义模型</span>
        </div>
        <div v-if="llmConfigs.length === 0" class="llm-empty">暂无模型配置，点击「新增」添加</div>
        <div v-else class="llm-list">
          <div v-for="(cfg, idx) in llmConfigs" :key="cfg.id" class="llm-item">
            <div class="llm-item-icon" :class="`llm-icon-${cfg.provider}`">
              <BrainCircuit :size="15" aria-hidden="true" />
            </div>
            <div class="llm-item-info">
              <div class="llm-item-name">
                {{ cfg.name }}
                <a-tag v-if="cfg.isDefault" color="green" style="margin-left: 4px; font-size: 11px">默认</a-tag>
              </div>
              <div class="llm-item-desc">{{ providerLabel(cfg.provider) }} · {{ cfg.model }}</div>
            </div>
            <div class="llm-item-actions">
              <a-button @click="doEditLLM(idx)">编辑</a-button>
              <a-button danger @click="doDeleteLLM(idx)">删除</a-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <LLMConfigModal v-model:open="showLLMModal" :initial="editingLLM" @save="onLLMSave" />
  </div>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue'
import { BrainCircuit, Plus, Sparkles } from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { main } from '../api/call'
import type { LLMConfig, LLMPXModelInfo, BuiltinModelSetting } from '../api/call'
import LLMConfigModal from './LLMConfigModal.vue'

const emit = defineEmits<{
  'open-login': []
}>()

const llmConfigs = ref<LLMConfig[]>([])
const showLLMModal = ref(false)
const editingLLM = ref<LLMConfig | null>(null)
const editingLLMIdx = ref(-1)

const builtinModels = ref<LLMPXModelInfo[]>([])
const builtinQuota = ref<number>(0)
const builtinApiUrl = ref('')
const builtinApiKey = ref('')
const builtinLoading = ref(false)
const builtinError = ref('')
const isLoggedIn = ref(false)
const showBuiltinSettingModal = ref(false)
const builtinModelSettings = ref<BuiltinModelSetting[]>([])

function getBuiltinModelVisible(name: string): boolean {
  const setting = builtinModelSettings.value.find((s) => s.name === name)
  return setting ? setting.visible : true
}

function getBuiltinModelDefault(name: string): boolean {
  const setting = builtinModelSettings.value.find((s) => s.name === name)
  return setting ? setting.isDefault : false
}

function ensureSetting(name: string): BuiltinModelSetting {
  let setting = builtinModelSettings.value.find((s) => s.name === name)
  if (!setting) {
    setting = { name, visible: true, isDefault: false }
    builtinModelSettings.value.push(setting)
  }
  return setting
}

async function onBuiltinVisibleChange(name: string, checked: boolean) {
  const setting = ensureSetting(name)
  setting.visible = checked
  await saveBuiltinSettings()
}

async function onBuiltinDefaultChange(name: string) {
  builtinModelSettings.value.forEach((s) => {
    s.isDefault = false
  })
  const updatedConfigs = llmConfigs.value.map((c) => ({
    ...c,
    isDefault: false,
  }))
  llmConfigs.value = updatedConfigs
  try {
    await main.Call('llm.saveLLMConfigs', { configs: updatedConfigs })
  } catch {}
  const setting = ensureSetting(name)
  setting.isDefault = true
  await saveBuiltinSettings()
}

async function saveBuiltinSettings() {
  try {
    await main.Call('llm.saveBuiltinModelSettings', {
      settings: builtinModelSettings.value,
    })
  } catch (e: any) {
    message.error(`保存失败：${e?.message || e}`)
  }
}

function formatQuota(q: number): string {
  if (q >= 1000000) return `${(q / 1000000).toFixed(1)}M`
  if (q >= 1000) return `${(q / 1000).toFixed(0)}K`
  return `${q}`
}

const providerLabel = (p: string) => ({ openai: 'OpenAI', gemini: 'Google Gemini', claude: 'Anthropic Claude' })[p] || p

async function checkLoginStatus() {
  try {
    const token = await main.Call('setting.getApiToken')
    isLoggedIn.value = !!token
  } catch {
    isLoggedIn.value = false
  }
}

async function loadBuiltinModels() {
  if (!isLoggedIn.value) return
  builtinLoading.value = true
  builtinError.value = ''
  try {
    const info = await main.Call('llm.getLLMPXInfo')
    builtinModels.value = info?.models || []
    builtinQuota.value = info?.quota || 0
    builtinApiUrl.value = info?.apiUrl || ''
    builtinApiKey.value = info?.apiKey || ''
  } catch (e: any) {
    builtinError.value = e?.message || '获取内置模型失败'
    builtinModels.value = []
    builtinQuota.value = 0
  } finally {
    builtinLoading.value = false
  }
}

async function loadBuiltinModelSettings() {
  try {
    builtinModelSettings.value = (await main.Call('llm.getBuiltinModelSettings')) || []
  } catch {}
}

async function loadLLMConfigs() {
  try {
    llmConfigs.value = (await main.Call('llm.getLLMConfigs')) || []
  } catch {}
}

function doRecharge() {
  if (!isLoggedIn.value) {
    emit('open-login')
    return
  }
  window.dispatchEvent(new CustomEvent('app:open-login', { detail: { page: 'ChargeLLMPX' } }))
}

async function onLoginSuccess() {
  await checkLoginStatus()
  await loadBuiltinModels()
  await loadBuiltinModelSettings()
}

function doAddLLM() {
  editingLLM.value = null
  editingLLMIdx.value = -1
  showLLMModal.value = true
}

function doEditLLM(idx: number) {
  editingLLM.value = { ...llmConfigs.value[idx] }
  editingLLMIdx.value = idx
  showLLMModal.value = true
}

async function doDeleteLLM(idx: number) {
  const list = [...llmConfigs.value]
  list.splice(idx, 1)
  try {
    await main.Call('llm.saveLLMConfigs', { configs: list })
    llmConfigs.value = list
    message.success('已删除')
  } catch (e: any) {
    message.error(`删除失败：${e?.message || e}`)
  }
}

async function onLLMSave(cfg: LLMConfig) {
  const list = [...llmConfigs.value]
  if (cfg.isDefault) {
    list.forEach((c) => {
      c.isDefault = false
    })
    // 同时取消内置模型的默认
  }
  if (editingLLMIdx.value >= 0) {
    list[editingLLMIdx.value] = cfg
  } else {
    list.push(cfg)
  }
  try {
    await main.Call('llm.saveLLMConfigs', { configs: list })
    llmConfigs.value = list
    message.success('已保存')
  } catch (e: any) {
    message.error(`保存失败：${e?.message || e}`)
  }
}

// 暴露给外部（ModelSelector 等组件使用）
const getBuiltinModelsComputed = computed(() => builtinModels.value)
const getBuiltinApiUrl = computed(() => builtinApiUrl.value)
const getBuiltinApiKeyComputed = computed(() => builtinApiKey.value)
const getIsLoggedIn = computed(() => isLoggedIn.value)

defineExpose({
  llmConfigs,
  getBuiltinModels: getBuiltinModelsComputed,
  getBuiltinApiUrl,
  getBuiltinApiKey: getBuiltinApiKeyComputed,
  getIsLoggedIn,
  loadLLMConfigs,
  loadBuiltinModels,
  onLoginSuccess,
})

onMounted(async () => {
  await checkLoginStatus()
  await loadLLMConfigs()
  await loadBuiltinModelSettings()
  if (isLoggedIn.value) {
    await loadBuiltinModels()
  }
  window.addEventListener('app:login-success', handleLoginSuccess)
})

onUnmounted(() => {
  window.removeEventListener('app:login-success', handleLoginSuccess)
})

function handleLoginSuccess() {
  onLoginSuccess()
}
</script>

<style scoped>
.llm-config-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 卡片 */
.card {
  background: #fff;
  border-radius: 12px;
  border: 1px solid #e5e5ea;
  overflow: hidden;
  padding-bottom: 4px;
}

.card-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #f2f2f7;
}

.section-title-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: #1c1c1e;
}

/* 内置模型横条 */
.builtin-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: linear-gradient(135deg, #f8f7ff 0%, #f0f4ff 100%);
}

.builtin-bar-left {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.builtin-bar-icon {
  color: #7c3aed;
}

.builtin-bar-label {
  font-size: 13px;
  font-weight: 600;
  color: #1c1c1e;
  white-space: nowrap;
}

.builtin-bar-center {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.builtin-bar-hint {
  font-size: 12px;
  color: #aeaeb2;
}

.builtin-bar-error {
  font-size: 12px;
  color: #ef4444;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.builtin-bar-quota {
  font-size: 12px;
  font-weight: 500;
  color: #7c3aed;
  white-space: nowrap;
}

.builtin-bar-divider {
  width: 1px;
  height: 12px;
  background: #d4d4d8;
  flex-shrink: 0;
}

.builtin-bar-models {
  font-size: 12px;
  color: #6b7280;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s;
}

.builtin-bar-models:hover {
  color: #7c3aed;
}

.builtin-bar-right {
  flex-shrink: 0;
}

/* 分隔线 */
.section-divider {
  height: 1px;
  background: #e5e5ea;
  margin: 0 16px;
}

/* 自定义模型区域 */
.custom-section {
  padding: 0;
}

.custom-section-header {
  padding: 10px 16px 4px;
}

.custom-section-label {
  font-size: 12px;
  font-weight: 600;
  color: #8e8e93;
}

/* 模型列表 */
.llm-empty {
  padding: 16px;
  font-size: 12px;
  color: #aeaeb2;
  text-align: center;
}

.llm-list {
  display: flex;
  flex-direction: column;
}

.llm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  border-bottom: 1px solid #f2f2f7;
}

.llm-item:last-child {
  border-bottom: none;
}

.llm-item-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  flex-shrink: 0;
}

.llm-icon-openai {
  background: linear-gradient(135deg, #018b8d, #037172);
}

.llm-icon-gemini {
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
}

.llm-icon-claude {
  background: linear-gradient(135deg, #f59e0b, #d97706);
}

.llm-item-info {
  flex: 1;
  min-width: 0;
}

.llm-item-name {
  font-size: 13px;
  font-weight: 500;
  color: #1c1c1e;
  display: flex;
  align-items: center;
}

.llm-item-desc {
  font-size: 11px;
  color: #8e8e93;
  margin-top: 2px;
}

.llm-item-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  align-items: center;
}

/* 内置模型设置弹窗 */
.builtin-setting-list {
  display: flex;
  flex-direction: column;
  gap: 0;
  max-height: 400px;
  overflow-y: auto;
}

.builtin-setting-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid #f2f2f7;
}

.builtin-setting-item:last-child {
  border-bottom: none;
}

.builtin-setting-name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  color: #1c1c1e;
}

.builtin-setting-icon {
  color: #7c3aed;
}

.builtin-setting-rate {
  font-size: 12px;
  color: #7c3aed;
  font-weight: 500;
}

.builtin-setting-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.builtin-modal-empty {
  padding: 24px;
  text-align: center;
  font-size: 12px;
  color: #aeaeb2;
}
</style>

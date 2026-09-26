<template>
  <a-modal
    :open="open"
    :title="isEdit ? '编辑模型' : '新增模型'"
    :width="'min(600px, 90vw)'"
    :mask-closable="false"
    ok-text="保存"
    cancel-text="取消"
    :confirm-loading="saving"
    @ok="doSave"
    @cancel="$emit('update:open', false)"
  >
    <a-form :model="form" layout="vertical" style="margin-top: 8px">
      <a-form-item label="名称" required>
        <a-input v-model:value="form.name" placeholder="如：GPT-4o、Claude 3.5" />
      </a-form-item>
      <a-form-item label="提供商" required>
        <a-select v-model:value="form.provider" @change="onProviderChange">
          <a-select-option value="openai">OpenAI</a-select-option>
          <a-select-option value="gemini">Google Gemini</a-select-option>
          <a-select-option value="claude">Anthropic Claude</a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item label="模型名称" required>
        <a-auto-complete
          v-model:value="form.model"
          :options="modelOptions"
          placeholder="如：gpt-4o、claude-sonnet-4-5"
          allow-clear
        />
      </a-form-item>
      <a-form-item label="API Key" required>
        <a-input-password v-model:value="form.apiKey" placeholder="sk-..." autocomplete="off" />
      </a-form-item>
      <a-form-item :label="`Base URL（可选，默认：${defaultBaseURL}）`">
        <a-input v-model:value="form.baseURL" :placeholder="defaultBaseURL" allow-clear />
      </a-form-item>
      <a-form-item>
        <a-checkbox v-model:checked="form.isDefault">设为默认模型</a-checkbox>
      </a-form-item>
      <a-form-item style="margin-bottom: 0">
        <div class="flex items-center gap-3">
          <a-button :loading="testing" @click="doTest">
            <div class="inline-flex items-center gap-1">
              <Zap class="w-4 h-4" aria-hidden="true" />
              测试连接
            </div>
          </a-button>
          <span
            v-if="testResult !== null"
            class="text-sm"
            :class="testResult.error ? 'text-red-500' : 'text-primary-600'"
          >
            {{ testResult.error ? `失败：${testResult.error}` : `成功：${testResult.content || 'OK'}` }}
          </span>
        </div>
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue'
import { Zap } from 'lucide-vue-next'
import { computed, reactive, ref, watch } from 'vue'
import { main } from '../api/call'
import type { LLMConfig, LLMCallResult } from '../api/call'

const props = defineProps<{
  open: boolean
  initial?: LLMConfig | null
}>()

const emit = defineEmits<{
  'update:open': [val: boolean]
  save: [cfg: LLMConfig]
}>()

const saving = ref(false)
const testing = ref(false)
const testResult = ref<LLMCallResult | null>(null)
const isEdit = computed(() => !!props.initial?.id)

const form = reactive<LLMConfig>({
  id: '',
  name: '',
  provider: 'openai',
  apiKey: '',
  baseURL: '',
  model: '',
  isDefault: false,
})

const providerModels: Record<string, string[]> = {
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-3.5-turbo', 'o1', 'o1-mini'],
  gemini: ['gemini-2.0-flash', 'gemini-2.0-flash-lite', 'gemini-1.5-pro', 'gemini-1.5-flash'],
  claude: ['claude-sonnet-4-5', 'claude-haiku-4-5', 'claude-opus-4-1', 'claude-3-5-sonnet-20241022'],
}

const providerBaseURL: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  gemini: 'https://generativelanguage.googleapis.com/v1beta/openai',
  claude: 'https://api.anthropic.com',
}

const modelOptions = computed(() => (providerModels[form.provider] || []).map((m) => ({ value: m })))
const defaultBaseURL = computed(() => providerBaseURL[form.provider] || '')

function onProviderChange() {
  const models = providerModels[form.provider]
  if (models?.length && !models.includes(form.model)) {
    form.model = models[0]
  }
}

watch(
  () => props.open,
  (val) => {
    if (val) {
      testResult.value = null
      if (props.initial) {
        Object.assign(form, props.initial)
      } else {
        Object.assign(form, {
          id: '',
          name: '',
          provider: 'openai',
          apiKey: '',
          baseURL: '',
          model: 'gpt-4o',
          isDefault: false,
        })
      }
    }
  }
)

async function doTest() {
  if (!form.apiKey.trim()) {
    message.warning('请填写 API Key')
    return
  }
  if (!form.model.trim()) {
    message.warning('请填写模型名称')
    return
  }
  testing.value = true
  testResult.value = null
  try {
    const result = await main.Call('llm.testLLMConfig', {
      provider: form.provider,
      apiKey: form.apiKey.trim(),
      baseURL: form.baseURL.trim(),
      model: form.model.trim(),
    })
    testResult.value = result
    if (result.error) {
      message.error(`测试失败：${result.error}`)
    } else {
      message.success('测试成功')
    }
  } catch (e: any) {
    testResult.value = { content: '', error: e?.message || '请求失败' }
    message.error(`测试失败：${e?.message || e}`)
  } finally {
    testing.value = false
  }
}

async function doSave() {
  if (!form.name.trim()) {
    message.warning('请填写名称')
    return
  }
  if (!form.model.trim()) {
    message.warning('请填写模型名称')
    return
  }
  if (!form.apiKey.trim()) {
    message.warning('请填写 API Key')
    return
  }
  saving.value = true
  try {
    const cfg: LLMConfig = {
      id: form.id || `llm-${Date.now()}`,
      name: form.name.trim(),
      provider: form.provider,
      apiKey: form.apiKey.trim(),
      baseURL: form.baseURL.trim(),
      model: form.model.trim(),
      isDefault: form.isDefault,
    }
    emit('save', cfg)
    emit('update:open', false)
  } finally {
    saving.value = false
  }
}
</script>

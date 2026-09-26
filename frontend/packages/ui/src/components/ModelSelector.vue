<template>
  <a-select
    :value="value"
    :disabled="disabled || loading"
    :placeholder="hasOptions ? t('browser.model.select') : t('browser.model.noModel')"
    allow-clear
    style="width: 100%"
    @update:value="emit('update:value', $event as string)"
  >
    <a-select-opt-group v-if="visibleBuiltinModels.length > 0" label="内置模型">
      <a-select-option v-for="m in visibleBuiltinModels" :key="'builtin:' + m.name" :value="'builtin:' + m.name">
        <div class="model-option">
          <ModelIcon :name="m.name" :provider="''" />
          <span class="model-label">{{ m.name }}</span>
          <span class="model-rate">×{{ m.rate }}</span>
        </div>
      </a-select-option>
    </a-select-opt-group>
    <a-select-opt-group v-if="configs.length > 0" label="自定义模型">
      <a-select-option v-for="c in configs" :key="c.id" :value="c.id">
        <div class="model-option">
          <ModelIcon :name="c.name" :provider="c.provider" />
          <span class="model-label">{{ c.name }}</span>
          <span v-if="c.isDefault" class="model-default">默认</span>
        </div>
      </a-select-option>
    </a-select-opt-group>
  </a-select>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { main } from '../api/call'
import type { LLMConfig } from '../api/call'

const { t } = useI18n()

// ── 模型 icon 组件 ────────────────────────────────────────────────────────────

// 根据 name/provider 匹配厂商信息
interface ProviderMeta {
  faviconUrl: string
  fallbackColor: string
  fallbackLetter: string
}

function resolveProvider(name: string, provider: string): ProviderMeta {
  const text = ((name || '') + ' ' + (provider || '')).toLowerCase()

  if (text.includes('openai') || text.includes('gpt') || text.includes('o1') || text.includes('o3')) {
    return {
      faviconUrl: 'https://openai.com/favicon.ico',
      fallbackColor: '#10a37f',
      fallbackLetter: 'O',
    }
  }
  if (text.includes('claude') || text.includes('anthropic')) {
    return {
      faviconUrl: 'https://claude.ai/favicon.ico',
      fallbackColor: '#d97706',
      fallbackLetter: 'C',
    }
  }
  if (text.includes('gemini') || text.includes('google') || text.includes('bard')) {
    return {
      faviconUrl: 'https://www.google.com/favicon.ico',
      fallbackColor: '#4285f4',
      fallbackLetter: 'G',
    }
  }
  if (text.includes('deepseek')) {
    return {
      faviconUrl: 'https://www.deepseek.com/favicon.ico',
      fallbackColor: '#2563eb',
      fallbackLetter: 'D',
    }
  }
  if (text.includes('qwen') || text.includes('tongyi') || text.includes('阿里') || text.includes('alibaba')) {
    return {
      faviconUrl: 'https://tongyi.aliyun.com/favicon.ico',
      fallbackColor: '#f97316',
      fallbackLetter: 'Q',
    }
  }
  if (text.includes('kimi') || text.includes('moonshot')) {
    return {
      faviconUrl: 'https://kimi.moonshot.cn/favicon.ico',
      fallbackColor: '#6366f1',
      fallbackLetter: 'K',
    }
  }
  if (text.includes('mistral')) {
    return {
      faviconUrl: 'https://mistral.ai/favicon.ico',
      fallbackColor: '#ef4444',
      fallbackLetter: 'M',
    }
  }
  if (text.includes('llama') || text.includes('meta')) {
    return {
      faviconUrl: 'https://ai.meta.com/favicon.ico',
      fallbackColor: '#0866ff',
      fallbackLetter: 'L',
    }
  }
  if (text.includes('ollama')) {
    return {
      faviconUrl: 'https://ollama.com/favicon.ico',
      fallbackColor: '#374151',
      fallbackLetter: 'O',
    }
  }
  if (text.includes('groq')) {
    return {
      faviconUrl: 'https://groq.com/favicon.ico',
      fallbackColor: '#f43f5e',
      fallbackLetter: 'G',
    }
  }
  if (text.includes('zhipu') || text.includes('chatglm') || text.includes('glm')) {
    return {
      faviconUrl: 'https://www.zhipuai.cn/favicon.ico',
      fallbackColor: '#7c3aed',
      fallbackLetter: 'Z',
    }
  }
  if (text.includes('baidu') || text.includes('ernie') || text.includes('wenxin')) {
    return {
      faviconUrl: 'https://www.baidu.com/favicon.ico',
      fallbackColor: '#2932e1',
      fallbackLetter: 'B',
    }
  }

  // 未知：用名称首字母
  const letter = (name || provider || '?').trim().charAt(0).toUpperCase()
  return { faviconUrl: '', fallbackColor: '#6b7280', fallbackLetter: letter }
}

// ModelIcon 子组件（内联定义，避免拆文件）
const ModelIcon = defineComponent({
  props: {
    name: { type: String, default: '' },
    provider: { type: String, default: '' },
  },
  setup(props) {
    const failed = ref(false)
    const meta = computed(() => resolveProvider(props.name, props.provider))

    function onError() {
      failed.value = true
    }

    return () => {
      const m = meta.value
      if (!failed.value && m.faviconUrl) {
        return h('img', {
          src: m.faviconUrl,
          class: 'model-icon-img',
          'aria-hidden': 'true',
          onError,
        })
      }
      // fallback：首字母色块
      return h(
        'span',
        {
          class: 'model-icon-fallback',
          style: { background: m.fallbackColor },
          'aria-hidden': 'true',
        },
        m.fallbackLetter
      )
    }
  },
})

// ── ModelSelector 主逻辑 ──────────────────────────────────────────────────────

interface BuiltinModel {
  name: string
  rate: number
}

interface BuiltinModelSetting {
  name: string
  visible: boolean
  isDefault: boolean
}

const props = defineProps<{
  value?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:value': [v: string]
}>()

const configs = ref<LLMConfig[]>([])
const builtinModels = ref<BuiltinModel[]>([])
const builtinSettings = ref<BuiltinModelSetting[]>([])
const loading = ref(false)

const visibleBuiltinModels = computed(() => {
  return builtinModels.value.filter((m) => {
    const setting = builtinSettings.value.find((s) => s.name === m.name)
    return setting ? setting.visible : true
  })
})

const hasOptions = computed(() => configs.value.length > 0 || visibleBuiltinModels.value.length > 0)

onMounted(async () => {
  loading.value = true
  try {
    configs.value = (await main.Call('llm.getLLMConfigs')) || []
    try {
      const info = await main.Call('llm.getLLMPXInfo')
      builtinModels.value = (info?.models || []).map((m: any) => ({
        name: m.name || m.id,
        rate: m.rate || 1,
      }))
    } catch {
      // 未登录或获取失败，不显示内置模型
    }
    try {
      builtinSettings.value = (await main.Call('llm.getBuiltinModelSettings')) || []
    } catch {}
    if (!props.value) {
      const defaultBuiltin = builtinSettings.value.find((s) => s.isDefault && s.visible)
      if (defaultBuiltin && visibleBuiltinModels.value.some((m) => m.name === defaultBuiltin.name)) {
        emit('update:value', 'builtin:' + defaultBuiltin.name)
      } else {
        const def = configs.value.find((c) => c.isDefault)
        if (def) {
          emit('update:value', def.id)
        } else if (visibleBuiltinModels.value.length > 0) {
          emit('update:value', 'builtin:' + visibleBuiltinModels.value[0].name)
        }
      }
    }
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.model-option {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

/* favicon 图片 */
.model-option :deep(.model-icon-img) {
  width: 14px;
  height: 14px;
  border-radius: 3px;
  flex-shrink: 0;
  object-fit: contain;
  display: block;
}

/* 加载失败时的首字母色块 */
.model-option :deep(.model-icon-fallback) {
  width: 14px;
  height: 14px;
  border-radius: 3px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 9px;
  font-weight: 700;
  color: #fff;
  line-height: 1;
}

.model-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.model-rate {
  font-size: 11px;
  color: #9ca3af;
  flex-shrink: 0;
}

.model-default {
  font-size: 10px;
  color: #fff;
  background: #3b82f6;
  border-radius: 3px;
  padding: 1px 5px;
  flex-shrink: 0;
  line-height: 1.4;
}
</style>

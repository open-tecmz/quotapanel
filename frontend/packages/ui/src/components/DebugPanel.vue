<script lang="ts" setup>
import { message } from 'ant-design-vue'
import { Globe, BookOpen, Copy, ChevronRight, ChevronLeft, Terminal } from 'lucide-vue-next'
import { computed, ref, watch, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useModuleConfig } from '../config/module'

const router = useRouter()
const route = useRoute()
const { moduleOptions, resolveModuleRoute } = useModuleConfig()

const iconMap: Record<string, Component> = { Globe, BookOpen }

const props = withDefaults(defineProps<{ open?: boolean }>(), { open: false })
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

// 支持 v-model:open（父组件控制）和本地点击箭头切换
const localOpen = ref(props.open)
const panelOpen = computed({
  get: () => localOpen.value,
  set: (val) => {
    localOpen.value = val
    emit('update:open', val)
  },
})

// 同步父组件传入的值
watch(
  () => props.open,
  (val) => {
    localOpen.value = val
  }
)

const currentModule = computed(() => {
  const found = moduleOptions.find((m) => route.path.startsWith('/' + m.value))
  return found ? found.value : (moduleOptions[0]?.value ?? '')
})

function onModuleSelect(val: string) {
  router.push(resolveModuleRoute(val))
}

function doCopyRoute() {
  const path = router.currentRoute.value.path
  navigator.clipboard
    .writeText(path)
    .then(() => {
      message.success(`已复制：${path}`)
    })
    .catch(() => {
      message.error('复制失败')
    })
}

function doOpenConsole() {
  // Wails v2 通过 window.WailsInvoke 向 Go 发消息
  // "wails:openInspector" 由 darwin/frontend.go 处理，调用 WKWebView._inspector.show
  const win = window as unknown as Record<string, unknown>
  const invoke = win['WailsInvoke'] as ((msg: string) => void) | undefined
  if (invoke) {
    invoke('wails:openInspector')
  } else {
    message.info('仅 dev/debug 构建支持，请右键「检查元素」')
  }
}
</script>

<template>
  <div class="debug-panel-wrap">
    <!-- 箭头触发按钮 -->
    <div class="debug-toggle-btn" :title="panelOpen ? '收起调试面板' : '展开调试面板'" @click="panelOpen = !panelOpen">
      <ChevronRight v-if="!panelOpen" class="w-3 h-3" aria-hidden="true" />
      <ChevronLeft v-else class="w-3 h-3" aria-hidden="true" />
    </div>
    <!-- 展开内容 -->
    <div v-if="panelOpen" class="debug-panel-body">
      <a-select :value="currentModule" @change="onModuleSelect">
        <a-select-option v-for="mod in moduleOptions" :key="mod.value" :value="mod.value">
          <component :is="iconMap[mod.iconName]" :size="13" class="inline mr-1" aria-hidden="true" />
          {{ mod.label }}
        </a-select-option>
      </a-select>
      <a-dropdown>
        <a-button>
          <div class="inline-flex items-center gap-1">
            <Terminal class="w-4 h-4" aria-hidden="true" />
            操作
          </div>
        </a-button>
        <template #overlay>
          <a-menu>
            <a-menu-item @click="doCopyRoute">
              <div class="inline-flex items-center gap-2">
                <Copy class="w-4 h-4" aria-hidden="true" />
                复制当前路由
              </div>
            </a-menu-item>
            <a-menu-item @click="doOpenConsole">
              <div class="inline-flex items-center gap-2">
                <Terminal class="w-4 h-4" aria-hidden="true" />
                打开 Console
              </div>
            </a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>
  </div>
</template>

<style scoped>
.debug-panel-wrap {
  position: fixed;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  z-index: 100;
  display: flex;
  align-items: center;
}

.debug-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 40px;
  background: #eeeeee;
  border: none;
  border-radius: 0 6px 6px 0;
  cursor: pointer;
  color: #111111;
  transition: background 0.15s;
  flex-shrink: 0;
  border: 1px solid #e5e7eb;
}

.debug-toggle-btn:hover {
  background: #ffffff;
}

.debug-panel-body {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-shadow: 2px 2px 8px rgba(0, 0, 0, 0.12);
}

:deep(.debug-panel-body .ant-select-selector) {
  background: #fff !important;
  border-color: #d9d9d9 !important;
  border-radius: 6px !important;
}
</style>

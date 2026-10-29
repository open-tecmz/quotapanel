<template>
  <div
    class="quota-card group flex flex-col bg-white rounded-xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow duration-200 cursor-pointer"
    :class="{
      'card-dragging': dragging,
      'card-drop-before': dropSide === 'before',
      'card-drop-after': dropSide === 'after',
    }"
    :data-account-id="account.id"
    draggable="true"
    @click="emit('detail')"
    @dragstart="onDragStart"
    @dragover.prevent="onDragOver"
    @dragend="onDragEnd"
  >
    <!-- 第一行：拖拽手柄 + 图标 + 名称 + 状态 -->
    <div class="flex items-center justify-between px-3 pt-2.5 pb-1 shrink-0 gap-2">
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <a-button
          type="text"
          class="card-handle shrink-0"
          :title="t('quota.account.reorder')"
          @mousedown="dragArmed = true"
          @mouseup="dragArmed = false"
          @click.stop
        >
          <GripVertical class="w-3.5 h-3.5" aria-hidden="true" />
        </a-button>
        <div
          class="w-7 h-7 shrink-0 rounded-lg flex items-center justify-center bg-primary-50 border border-primary-100"
        >
          <Wallet class="w-3.5 h-3.5 text-primary-500" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <div class="text-[13px] font-semibold text-gray-900 truncate">
            {{ account.name }}
          </div>
          <div class="card-subtitle text-[11px] text-gray-400 truncate">
            {{ subtitle }}
          </div>
        </div>
      </div>
      <a-tag :color="tagColor" class="!mr-0 shrink-0">{{ tagLabel }}</a-tag>
    </div>

    <!-- 中间行：备注 / Key（紧凑单行） -->
    <div class="px-3 pb-1 flex items-center gap-1 text-[11px] text-gray-400 min-w-0">
      <span v-if="account.note" class="card-note truncate" :title="account.note">{{ noteFirstLine }}</span>
      <span v-if="account.note && !browserMode" class="shrink-0 text-gray-300">·</span>
      <span v-if="!browserMode" class="font-mono truncate" :title="account.keyMasked">{{ account.keyMasked }}</span>
      <a-button v-if="!browserMode" type="text" :title="t('quota.account.copy')" @click.stop="emit('copy')">
        <Copy class="w-3 h-3 text-gray-400" aria-hidden="true" />
      </a-button>
    </div>

    <!-- 额度详情（默认仅展示 3 条进度条，其余在详情弹窗中查看） -->
    <div class="card-details px-3 pb-2">
      <div
        v-if="snapshot && snapshot.error"
        class="text-[11px] text-red-500 bg-red-50 rounded-lg px-2.5 py-1.5 break-all"
      >
        {{ snapshot.error }}
      </div>
      <template v-else-if="snapshot">
        <div v-if="visibleWindows.length" class="window-grid">
          <template v-for="w in visibleWindows" :key="w.key">
            <span class="window-label" :title="w.label">{{ w.label }}</span>
            <a-progress
              class="window-bar"
              :percent="Math.min(100, Math.round(w.percent))"
              :stroke-color="progressColor(w.status)"
              :show-info="false"
              size="small"
            />
            <span class="window-value" :title="w.detail || `${w.percent.toFixed(0)}%`">
              {{ w.detail || `${w.percent.toFixed(0)}%` }}
            </span>
            <span
              v-if="hasResetTime(w.resetAt)"
              class="window-reset"
              :title="`${t('quota.account.resetAt')}：${w.resetAt}`"
            >
              <RotateCcw class="w-3 h-3 shrink-0" aria-hidden="true" />
              <span class="window-reset-text">{{ w.resetAt }}</span>
            </span>
            <span v-else class="window-reset-empty" aria-hidden="true" />
          </template>
          <div v-if="hiddenWindowCount" class="window-more">
            {{ t('quota.account.moreWindows', { count: hiddenWindowCount }) }}
          </div>
        </div>
        <div v-else-if="previewStats.length" class="space-y-0.5">
          <div
            v-for="s in previewStats"
            :key="s.label"
            class="flex items-center justify-between text-[11px]"
            :class="s.muted ? 'text-gray-400' : 'text-gray-600'"
          >
            <span class="truncate">{{ s.label }}</span>
            <span class="font-medium truncate ml-2" :title="s.value">{{ s.value }}</span>
          </div>
          <div v-if="hiddenStatCount" class="text-[10px] text-primary-600">
            {{ t('quota.account.moreWindows', { count: hiddenStatCount }) }}
          </div>
        </div>
        <div v-else class="text-[11px] text-gray-300 py-1 text-center">
          {{ t('quota.account.detailEmpty') }}
        </div>
      </template>
      <div v-else class="text-[11px] text-gray-300 py-1 text-center">
        {{ loading ? t('quota.account.loading') : t('quota.account.queryHint') }}
      </div>
    </div>

    <!-- 最后一行：更新时间 + 操作（操作默认隐藏，hover 时显示） -->
    <div class="px-3 pb-2 pt-1 shrink-0 text-xs border-t border-gray-50">
      <div class="flex items-center justify-between gap-2">
        <span
          class="flex items-center gap-1 min-w-0 text-[10px] text-gray-400"
          :title="`#${account.id} · ${updateTimeText}`"
        >
          <Clock class="w-3 h-3 shrink-0" aria-hidden="true" />
          <span class="truncate">{{ updateTimeValue }}</span>
        </span>
        <div
          class="card-actions flex items-center justify-end shrink-0 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity duration-150"
          @click.stop
        >
          <a-button v-if="browserMode" type="text" :title="t('quota.account.login')" @click.stop="emit('login')">
            <Globe class="w-4 h-4 text-gray-500" aria-hidden="true" />
          </a-button>
          <a-button v-if="browserMode" type="text" :title="t('quota.account.verify')" @click.stop="emit('verify')">
            <CircleCheck class="w-4 h-4 text-primary-600" aria-hidden="true" />
          </a-button>
          <a-button type="text" :title="t('quota.account.refresh')" @click.stop="emit('refresh')">
            <RefreshCw class="w-4 h-4 text-gray-500" :class="{ 'animate-spin': loading }" aria-hidden="true" />
          </a-button>
          <a-dropdown :trigger="['click']" placement="bottomRight">
            <a-button type="text" :title="t('quota.account.more')">
              <MoreVertical class="w-4 h-4 text-gray-500" aria-hidden="true" />
            </a-button>
            <template #overlay>
              <a-menu @click="onActionSelect">
                <a-menu-item key="share">
                  <span class="inline-flex items-center gap-2">
                    <Share2 class="w-4 h-4" aria-hidden="true" />
                    {{ t('quota.account.share') }}
                  </span>
                </a-menu-item>
                <a-menu-item key="edit">
                  <span class="inline-flex items-center gap-2">
                    <Pencil class="w-4 h-4" aria-hidden="true" />
                    {{ t('quota.account.edit') }}
                  </span>
                </a-menu-item>
                <a-menu-item key="delete" danger>
                  <span class="inline-flex items-center gap-2">
                    <Trash2 class="w-4 h-4" aria-hidden="true" />
                    {{ t('quota.account.delete') }}
                  </span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  CircleCheck,
  Clock,
  Copy,
  Globe,
  GripVertical,
  MoreVertical,
  Pencil,
  RefreshCw,
  RotateCcw,
  Share2,
  Trash2,
  Wallet,
} from 'lucide-vue-next'
import type { QuotaAccount, QuotaSnapshot } from '../../api/main'
import {
  aggregateStatus,
  firstLine,
  formatDateTime,
  hasResetTime,
  statusColor,
  statusTagColor,
  statusTextKey,
} from '../../utils/format'

const { t } = useI18n()

const props = defineProps<{
  account: QuotaAccount
  snapshot?: QuotaSnapshot
  loading: boolean
  browserMode: boolean
  dragging: boolean
  dropSide: 'before' | 'after' | null
}>()

const emit = defineEmits<{
  refresh: []
  edit: []
  delete: []
  copy: []
  share: []
  login: []
  verify: []
  detail: []
  dragStart: []
  dragOver: [after: boolean]
  dragEnd: []
}>()

// 仅当用户按下拖拽手柄后才允许开始拖拽，避免误拖卡片或卡片内的按钮。
const dragArmed = ref(false)

function onDragStart(event: DragEvent) {
  if (!dragArmed.value) {
    event.preventDefault()
    return
  }
  emit('dragStart')
}

// 用指针在卡片上的左右半区判断插入方向，拖拽过程只在松手时落位。
function onDragOver(event: DragEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  emit('dragOver', event.clientX > rect.left + rect.width / 2)
}

function onDragEnd() {
  dragArmed.value = false
  emit('dragEnd')
}

const updateTimeText = computed(() =>
  props.snapshot?.updatedAt
    ? t('quota.account.updatedAt', { time: formatDateTime(props.snapshot.updatedAt) })
    : t('quota.account.accountUpdatedAt', { time: formatDateTime(props.account.updatedAt) })
)

const updateTimeValue = computed(() => formatDateTime(props.snapshot?.updatedAt || props.account.updatedAt))

// 卡片只展示备注首行，完整备注在详情弹窗中查看。
const noteFirstLine = computed(() => firstLine(props.account.note))

// 供应商与套餐同名时只显示一次，避免卡片内出现重复文本。
const subtitle = computed(() => {
  const provider = props.account.providerName?.trim() || ''
  const plan = props.snapshot?.plan?.trim() || ''
  if (!plan || plan === provider) return provider
  return `${provider} · ${plan}`
})

// 卡片保持紧凑：额度窗口最多展示 3 条，其余进入详情弹窗
const MAX_WINDOWS = 3
const visibleWindows = computed(() => (props.snapshot?.windows || []).slice(0, MAX_WINDOWS))
const hiddenWindowCount = computed(() => Math.max(0, (props.snapshot?.windows?.length || 0) - MAX_WINDOWS))

// 无额度窗口时，用最多 3 行明细概览余额 / 统计
const previewStats = computed(() => {
  if ((props.snapshot?.windows?.length || 0) > 0) return []
  return [...(props.snapshot?.balances || []), ...(props.snapshot?.stats || [])].slice(0, MAX_WINDOWS)
})
const hiddenStatCount = computed(() => {
  if ((props.snapshot?.windows?.length || 0) > 0) return 0
  const total = (props.snapshot?.balances?.length || 0) + (props.snapshot?.stats?.length || 0)
  return Math.max(0, total - MAX_WINDOWS)
})

const currentStatus = computed(() => {
  if (props.snapshot?.error) return 'exceeded'
  return aggregateStatus(props.snapshot?.status, props.snapshot?.windows)
})

const tagColor = computed(() => statusTagColor(currentStatus.value))

const tagLabel = computed(() => {
  if (!props.snapshot) return t('quota.account.notQueried')
  return t(`quota.account.${statusTextKey(currentStatus.value)}`)
})

function progressColor(status: string): string {
  return statusColor(status)
}

function onActionSelect(info: { key: string | number }) {
  const key = String(info.key)
  if (key === 'share') emit('share')
  else if (key === 'edit') emit('edit')
  else if (key === 'delete') emit('delete')
}
</script>

<style scoped>
.quota-card {
  min-height: 140px;
  height: auto;
  align-self: stretch;
  position: relative;
  /* WebKit（WKWebView）需要显式声明可拖拽区域 */
  -webkit-user-drag: element;
}

/* 拖拽进行中：源卡片半透明，明确当前拖动对象 */
.card-dragging {
  opacity: 0.5;
}

/* 落位指示条：拖拽到某张卡片左右半区时提示插入位置 */
.card-drop-before::before,
.card-drop-after::after {
  content: '';
  position: absolute;
  top: 6px;
  bottom: 6px;
  width: 3px;
  border-radius: 2px;
  background: var(--token-primary-600);
  z-index: 2;
}
.card-drop-before::before {
  left: -7px;
}
.card-drop-after::after {
  right: -7px;
}

/* 拖拽手柄：默认隐藏，悬停卡片时显示，按住可拖拽排序 */
.card-handle {
  opacity: 0;
  cursor: grab;
  transition: opacity 0.15s;
}
.quota-card:hover .card-handle,
.card-handle:focus-visible {
  opacity: 1;
}
.card-handle:active {
  cursor: grabbing;
}
/* 按钮本身即 .card-handle，用双类提升优先级覆盖 antd 默认内边距 */
.quota-card .card-handle {
  padding: 0 2px;
  height: 20px;
  min-width: 16px;
}
.card-handle :deep(.anticon) {
  color: #9ca3af;
}

.card-details {
  flex: 1 1 auto;
}

/* 卡片操作按钮：更紧凑，且默认隐藏、hover 时显示 */
.card-actions :deep(.ant-btn) {
  padding: 0 3px;
  height: 22px;
  min-width: 22px;
}
.card-actions :deep(.ant-btn + .ant-btn) {
  margin-left: 0;
}

/* 额度窗口使用网格对齐：标签列与数值列自适应同一宽度，进度条左右边界对齐，文字不换行不截断 */
.window-grid {
  display: grid;
  grid-template-columns: minmax(0, auto) minmax(0, 1fr) auto auto;
  column-gap: 8px;
  row-gap: 4px;
  align-items: center;
}

.window-label {
  font-size: 11px;
  line-height: 1.4;
  color: #6b7280;
  max-width: 96px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.window-bar {
  min-width: 0;
}

.window-value {
  font-size: 11px;
  line-height: 1.4;
  color: #6b7280;
  text-align: right;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

/* 重置时间与进度条同行展示，仅保留图标 + 时间，完整文案在 title 中 */
.window-reset {
  display: flex;
  align-items: center;
  gap: 3px;
  min-width: 0;
  font-size: 10px;
  line-height: 1.3;
  color: #9ca3af;
  white-space: nowrap;
}

.window-reset-text {
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-variant-numeric: tabular-nums;
}

.window-more {
  grid-column: 1 / -1;
  font-size: 10px;
  color: var(--token-primary-600);
}

:global(.dark .quota-card .window-label),
:global(.dark .quota-card .window-value) {
  color: #d1d5db;
}

:global(.dark .quota-card) {
  background: var(--token-panel);
  border-color: #374151;
}
:global(.dark .quota-card .text-gray-900) {
  color: #f9fafb;
}
:global(.dark .quota-card .text-gray-600),
:global(.dark .quota-card .text-gray-500) {
  color: #d1d5db;
}
:global(.dark .quota-card .text-gray-400),
:global(.dark .quota-card .text-gray-300) {
  color: #9ca3af;
}
:global(.dark .quota-card .border-gray-50) {
  border-color: #374151;
}
</style>

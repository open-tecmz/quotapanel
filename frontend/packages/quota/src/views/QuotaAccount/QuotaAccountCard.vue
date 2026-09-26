<template>
  <div
    class="quota-card group flex flex-col bg-white rounded-xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow duration-200 cursor-pointer"
    :data-account-id="account.id"
    @click="emit('detail')"
  >
    <!-- 第一行：图标 + 名称 + 状态 -->
    <div class="flex items-center justify-between px-3 pt-2.5 pb-1 shrink-0 gap-2">
      <div class="flex items-center gap-2 min-w-0 flex-1">
        <div
          class="w-7 h-7 shrink-0 rounded-lg flex items-center justify-center bg-primary-50 border border-primary-100"
        >
          <Wallet class="w-3.5 h-3.5 text-primary-500" aria-hidden="true" />
        </div>
        <div class="min-w-0">
          <div class="text-[13px] font-semibold text-gray-900 truncate">
            {{ account.name }}
          </div>
          <div class="text-[11px] text-gray-400 truncate">
            {{ account.providerName }}
          </div>
        </div>
      </div>
      <a-tag :color="tagColor" class="!mr-0 shrink-0">{{ tagLabel }}</a-tag>
    </div>

    <!-- 中间行：备注 / Key（紧凑单行） -->
    <div class="px-3 pb-1 flex items-center gap-1 text-[11px] text-gray-400 min-w-0">
      <span v-if="account.note" class="truncate" :title="account.note">{{ account.note }}</span>
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
        <div v-if="snapshot.plan" class="text-[11px] text-gray-500 mb-1 truncate">{{ snapshot.plan }}</div>
        <div v-if="visibleWindows.length" class="window-grid">
          <template v-for="w in visibleWindows" :key="w.key">
            <span
              class="window-label"
              :title="w.resetAt ? `${w.label} · ${t('quota.account.resetAt')}: ${w.resetAt}` : w.label"
            >
              {{ w.label }}
            </span>
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
                <a-menu-item key="screenshot">
                  <span class="inline-flex items-center gap-2">
                    <Camera class="w-4 h-4" aria-hidden="true" />
                    {{ t('quota.account.screenshot') }}
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
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Camera,
  CircleCheck,
  Clock,
  Copy,
  Globe,
  MoreVertical,
  Pencil,
  RefreshCw,
  Trash2,
  Wallet,
} from 'lucide-vue-next'
import type { QuotaAccount, QuotaSnapshot } from '../../api/main'
import { formatDateTime, statusColor, statusTagColor, statusTextKey } from '../../utils/format'

const { t } = useI18n()

const props = defineProps<{
  account: QuotaAccount
  snapshot?: QuotaSnapshot
  loading: boolean
  browserMode: boolean
}>()

const emit = defineEmits<{
  refresh: []
  edit: []
  delete: []
  copy: []
  screenshot: []
  login: []
  verify: []
  detail: []
}>()

const updateTimeText = computed(() =>
  props.snapshot?.updatedAt
    ? t('quota.account.updatedAt', { time: formatDateTime(props.snapshot.updatedAt) })
    : t('quota.account.accountUpdatedAt', { time: formatDateTime(props.account.updatedAt) })
)

const updateTimeValue = computed(() => formatDateTime(props.snapshot?.updatedAt || props.account.updatedAt))

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
  return props.snapshot?.status
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
  if (key === 'screenshot') emit('screenshot')
  else if (key === 'edit') emit('edit')
  else if (key === 'delete') emit('delete')
}
</script>

<style scoped>
.quota-card {
  min-height: 140px;
  height: auto;
  align-self: stretch;
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
  grid-template-columns: minmax(0, auto) minmax(0, 1fr) auto;
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

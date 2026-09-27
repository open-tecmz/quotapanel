<template>
  <a-modal
    :open="open"
    :title="account?.name || t('quota.account.detailTitle')"
    width="min(720px, 90vw)"
    @cancel="handleCancel"
  >
    <div v-if="account" class="quota-detail">
      <!-- 概览：供应商 / 套餐 / 状态 / 账号 -->
      <div class="detail-head">
        <div class="flex items-center justify-between gap-3 min-w-0">
          <div class="min-w-0">
            <div class="text-[13px] font-semibold text-gray-900 truncate">{{ account.name }}</div>
            <div class="text-[11px] text-gray-400 truncate mt-0.5">
              {{ account.providerName }}
              <template v-if="snapshot?.plan"> · {{ snapshot.plan }}</template>
            </div>
          </div>
          <a-tag :color="tagColor" class="!mr-0 shrink-0">{{ tagLabel }}</a-tag>
        </div>
        <div v-if="snapshot?.accountName" class="text-[11px] text-gray-500 truncate mt-1">
          {{ snapshot.accountName }}
        </div>
        <div v-if="account.note" class="text-[11px] text-gray-500 truncate mt-1">{{ account.note }}</div>
        <div v-if="account.keyMasked" class="font-mono text-[11px] text-gray-400 truncate mt-1">
          {{ account.keyMasked }}
        </div>
      </div>

      <div v-if="snapshot?.error" class="detail-error">{{ snapshot.error }}</div>

      <template v-else-if="snapshot">
        <!-- 额度窗口：展示该订阅的每一种额度窗口 -->
        <section v-if="snapshot.windows.length" class="detail-section">
          <div class="section-title">
            <Gauge class="w-3.5 h-3.5 text-primary-500" aria-hidden="true" />
            {{ t('quota.account.windows') }}
          </div>
          <div class="space-y-3">
            <div v-for="w in snapshot.windows" :key="w.key" class="detail-window">
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs text-gray-600 truncate">{{ w.label }}</span>
                <span class="text-xs text-gray-500 shrink-0">{{ w.detail || `${w.percent.toFixed(0)}%` }}</span>
              </div>
              <a-progress
                :percent="Math.min(100, Math.round(w.percent))"
                :stroke-color="progressColor(w.status)"
                :show-info="false"
                size="small"
              />
              <div v-if="hasResetTime(w.resetAt)" class="text-[11px] text-gray-400 truncate mt-0.5">
                {{ t('quota.account.resetAt') }}：{{ w.resetAt }}
              </div>
            </div>
          </div>
        </section>

        <!-- 余额明细 -->
        <section v-if="snapshot.balances.length" class="detail-section">
          <div class="section-title">
            <Coins class="w-3.5 h-3.5 text-primary-500" aria-hidden="true" />
            {{ t('quota.account.balances') }}
          </div>
          <div class="stat-list">
            <div
              v-for="b in snapshot.balances"
              :key="b.label"
              class="detail-balance-row"
              :class="b.muted ? 'text-gray-400' : 'text-gray-600'"
            >
              <span class="truncate">{{ b.label }}</span>
              <span class="font-medium shrink-0 ml-3">{{ b.value }}</span>
            </div>
          </div>
        </section>

        <!-- 用量统计 -->
        <section v-if="snapshot.stats.length" class="detail-section">
          <div class="section-title">
            <BarChart3 class="w-3.5 h-3.5 text-primary-500" aria-hidden="true" />
            {{ t('quota.account.stats') }}
          </div>
          <div class="stat-list">
            <div v-for="s in snapshot.stats" :key="s.label" class="detail-stat-row text-gray-500">
              <span class="truncate">{{ s.label }}</span>
              <span class="font-medium shrink-0 ml-3 max-w-[60%] truncate" :title="s.value">{{ s.value }}</span>
            </div>
          </div>
        </section>

        <div v-if="snapshot.summary" class="detail-summary">{{ snapshot.summary }}</div>

        <div
          v-if="!snapshot.windows.length && !snapshot.balances.length && !snapshot.stats.length && !snapshot.summary"
          class="detail-empty"
        >
          {{ t('quota.account.detailEmpty') }}
        </div>
      </template>

      <div v-else class="detail-empty">
        {{ loading ? t('quota.account.loading') : t('quota.account.queryHint') }}
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-between gap-3">
        <span class="text-[11px] text-gray-400 truncate">{{ updateTimeText }}</span>
        <a-button @click="handleCancel">{{ t('quota.account.close') }}</a-button>
      </div>
    </template>
  </a-modal>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { BarChart3, Coins, Gauge } from 'lucide-vue-next'
import type { QuotaAccount, QuotaSnapshot } from '../../api/main'
import { formatDateTime, hasResetTime, statusColor, statusTagColor, statusTextKey } from '../../utils/format'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  account?: QuotaAccount | null
  snapshot?: QuotaSnapshot
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

function handleCancel() {
  emit('update:open', false)
}

const updateTimeText = computed(() => {
  if (props.snapshot?.updatedAt) return t('quota.account.updatedAt', { time: formatDateTime(props.snapshot.updatedAt) })
  if (props.account?.updatedAt)
    return t('quota.account.accountUpdatedAt', { time: formatDateTime(props.account.updatedAt) })
  return '-'
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
</script>

<style scoped>
.quota-detail {
  overflow-x: hidden;
}

.detail-head {
  padding: 12px;
  border-radius: 12px;
  background: #f8fafc;
  border: 1px solid #eef2f6;
}

.detail-error {
  margin-top: 12px;
  font-size: 12px;
  color: #dc2626;
  background: #fef2f2;
  border-radius: 10px;
  padding: 10px 12px;
  word-break: break-all;
  overflow-wrap: anywhere;
}

.detail-section {
  margin-top: 16px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 8px;
}

.stat-list {
  border-radius: 12px;
  border: 1px solid #eef2f6;
  overflow: hidden;
}

.detail-balance-row,
.detail-stat-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  padding: 8px 12px;
  background: #fff;
}

.detail-balance-row + .detail-balance-row,
.detail-stat-row + .detail-stat-row {
  border-top: 1px solid #f3f4f6;
}

.detail-summary {
  margin-top: 16px;
  font-size: 12px;
  color: var(--token-primary-600);
  background: #f0f9f9;
  border-radius: 10px;
  padding: 10px 12px;
  word-break: break-word;
}

.detail-empty {
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
  padding: 24px 0;
}

:global(.dark .quota-detail .detail-head) {
  background: #111827;
  border-color: #374151;
}
:global(.dark .quota-detail .text-gray-900) {
  color: #f9fafb;
}
:global(.dark .quota-detail .text-gray-600),
:global(.dark .quota-detail .text-gray-500) {
  color: #d1d5db;
}
:global(.dark .quota-detail .text-gray-400) {
  color: #9ca3af;
}
:global(.dark .quota-detail .section-title) {
  color: #e5e7eb;
}
:global(.dark .quota-detail .stat-list) {
  border-color: #374151;
}
:global(.dark .quota-detail .detail-balance-row),
:global(.dark .quota-detail .detail-stat-row) {
  background: #1f2937;
}
:global(.dark .quota-detail .detail-balance-row + .detail-balance-row),
:global(.dark .quota-detail .detail-stat-row + .detail-stat-row) {
  border-top-color: #374151;
}
:global(.dark .quota-detail .detail-summary) {
  background: rgba(var(--token-primary-rgb), 0.14);
  color: var(--token-primary-700);
}
</style>

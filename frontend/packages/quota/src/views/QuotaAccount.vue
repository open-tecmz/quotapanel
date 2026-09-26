<template>
  <div class="quota-view">
    <div class="view-header">
      <PageHeader :title="t('quota.nav.account')">
        <template #icon>
          <Wallet :size="22" class="text-primary-500" aria-hidden="true" />
        </template>
        <template #extra>
          <a-button :loading="queryingAll" :disabled="accounts.length === 0" @click="doQueryAll">
            <div class="inline-flex items-center gap-1">
              <RefreshCw class="w-4 h-4" aria-hidden="true" />
              {{ t('quota.account.refresh') }}
            </div>
          </a-button>
          <a-button type="primary" @click="doAdd">
            <div class="inline-flex items-center gap-1">
              <Plus class="w-4 h-4" aria-hidden="true" />
              {{ t('quota.account.add') }}
            </div>
          </a-button>
        </template>
      </PageHeader>
    </div>

    <!-- 空状态 -->
    <div v-if="accounts.length === 0 && !loadingAccounts" class="empty-state">
      <Wallet :size="48" class="empty-icon" aria-hidden="true" />
      <p class="empty-title">{{ t('quota.account.empty') }}</p>
      <p class="empty-desc">{{ t('quota.account.emptyDesc') }}</p>
      <a-button type="primary" @click="doAdd">
        <div class="inline-flex items-center gap-1">
          <Plus class="w-4 h-4" aria-hidden="true" />
          {{ t('quota.account.addSubscription') }}
        </div>
      </a-button>
    </div>

    <!-- 卡片列表 -->
    <div v-else class="account-list">
      <QuotaAccountCard
        v-for="account in accounts"
        :key="account.id"
        :account="account"
        :snapshot="snapshots[account.id]"
        :loading="queryingIds.has(account.id)"
        :browser-mode="providers.some((p) => p.id === account.provider && p.mode === 'browser')"
        @refresh="doQueryAccount(account.id)"
        @edit="doEdit(account)"
        @delete="doDelete(account)"
        @copy="doCopy(account)"
        @screenshot="doScreenshot(account)"
        @login="doBeginLogin(account.id)"
        @verify="doCompleteLogin(account.id)"
        @detail="doDetail(account)"
      />
    </div>

    <QuotaAccountModal
      ref="modalRef"
      v-model:open="showModal"
      :providers="providers"
      :account="editingAccount"
      @saved="onSaved"
    />

    <QuotaAccountDetailModal
      v-model:open="showDetail"
      :account="detailAccount"
      :snapshot="detailAccount ? snapshots[detailAccount.id] : undefined"
      :loading="detailAccount ? queryingIds.has(detailAccount.id) : false"
    />
  </div>
</template>

<script setup lang="ts">
import { message, Modal } from 'ant-design-vue'
import { Plus, RefreshCw, Wallet } from 'lucide-vue-next'
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { EventsOn, EventsOff } from '@quotapanel/ui/wailsjs/runtime/runtime'
import PageHeader from '@quotapanel/ui/src/components/PageHeader.vue'
import { testActionSet, testActionUnset } from '@quotapanel/ui/src/utils/test'
import { useAppStore } from '@quotapanel/ui/src/stores/app'
import type { QuotaAccount, QuotaProviderInfo, QuotaSnapshot } from '../api/main'
import { main } from '../api/main'
import QuotaAccountCard from './QuotaAccount/QuotaAccountCard.vue'
import QuotaAccountDetailModal from './QuotaAccount/QuotaAccountDetailModal.vue'
import QuotaAccountModal from './QuotaAccount/QuotaAccountModal.vue'
import { renderCardScreenshot } from '../utils/cardScreenshot'

const { t } = useI18n()
const appStore = useAppStore()
const router = useRouter()

const providers = ref<QuotaProviderInfo[]>([])
const accounts = ref<QuotaAccount[]>([])
const snapshots = ref<Record<number, QuotaSnapshot>>({})
const queryingIds = ref<Set<number>>(new Set())
const loadingAccounts = ref(false)
const queryingAll = ref(false)

const showModal = ref(false)
const editingAccount = ref<QuotaAccount | null>(null)
const modalRef = ref<InstanceType<typeof QuotaAccountModal> | null>(null)

const showDetail = ref(false)
const detailAccount = ref<QuotaAccount | null>(null)

async function loadAccounts() {
  loadingAccounts.value = true
  try {
    accounts.value = (await main.Call('quota.getAccounts')) || []
  } finally {
    loadingAccounts.value = false
  }
}

async function loadProviders() {
  providers.value = (await main.Call('quota.getProviders')) || []
}

async function doQueryAll() {
  if (accounts.value.length === 0) return
  queryingAll.value = true
  try {
    const list = (await main.Call('quota.queryAll')) || []
    const next: Record<number, QuotaSnapshot> = { ...snapshots.value }
    for (const snap of list) {
      next[snap.accountId] = snap
    }
    snapshots.value = next
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    queryingAll.value = false
  }
}

async function doQueryAccount(id: number) {
  queryingIds.value = new Set(queryingIds.value).add(id)
  try {
    const snap = await main.Call('quota.queryAccount', { id })
    snapshots.value = { ...snapshots.value, [id]: snap }
  } catch (e: unknown) {
    snapshots.value = {
      ...snapshots.value,
      [id]: {
        accountId: id,
        provider: '',
        accountName: '',
        plan: '',
        status: '',
        windows: [],
        balances: [],
        stats: [],
        summary: '',
        updatedAt: '',
        error: e instanceof Error ? e.message : String(e),
      },
    }
  } finally {
    const next = new Set(queryingIds.value)
    next.delete(id)
    queryingIds.value = next
  }
}

function doAdd() {
  editingAccount.value = null
  showModal.value = true
}

function doEdit(account: QuotaAccount) {
  editingAccount.value = account
  showModal.value = true
}

function doDetail(account: QuotaAccount) {
  detailAccount.value = account
  showDetail.value = true
}

function doDelete(account: QuotaAccount) {
  Modal.confirm({
    title: t('quota.account.deleteTitle'),
    content: t('quota.account.deleteConfirm', { name: account.name }),
    okText: t('quota.account.delete'),
    okType: 'danger',
    cancelText: t('quota.account.cancel'),
    onOk: async () => {
      try {
        await main.Call('quota.deleteAccount', { id: account.id })
        const next = { ...snapshots.value }
        delete next[account.id]
        snapshots.value = next
        await loadAccounts()
        message.success(t('quota.account.deleted'))
      } catch (e: unknown) {
        message.error(e instanceof Error ? e.message : String(e))
      }
    },
  })
}

async function doCopy(account: QuotaAccount) {
  try {
    await navigator.clipboard.writeText(account.key)
    message.success(t('quota.account.copied'))
  } catch {
    message.error(t('quota.account.copyFailed'))
  }
}

async function doScreenshot(account: QuotaAccount) {
  try {
    const pngBase64 = renderCardScreenshot(account, snapshots.value[account.id], appStore.locale)
    const saved = await main.Call('quota.saveScreenshot', { accountId: account.id, pngBase64 })
    if (saved) message.success(t('quota.account.screenshotSaved'))
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function doBeginLogin(id: number) {
  try {
    await main.Call('quota.beginLogin', { id })
    message.info(t('quota.account.loginHint'))
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function doCompleteLogin(id: number) {
  queryingIds.value = new Set(queryingIds.value).add(id)
  try {
    const snap = await main.Call('quota.completeLogin', { id })
    snapshots.value = { ...snapshots.value, [id]: snap }
    message.success(t('quota.account.loginVerified'))
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    const next = new Set(queryingIds.value)
    next.delete(id)
    queryingIds.value = next
  }
}

async function onSaved(account: QuotaAccount) {
  await loadAccounts()
  if (providers.value.some((p) => p.id === account.provider && p.mode === 'browser')) {
    await doBeginLogin(account.id)
  } else await doQueryAccount(account.id)
}

async function init() {
  await loadProviders()
  await loadAccounts()
  await doQueryAll()
}

onMounted(() => {
  init()
  EventsOn('quota:snapshots', (list: QuotaSnapshot[]) => {
    const next = { ...snapshots.value }
    for (const snap of list || []) next[snap.accountId] = snap
    snapshots.value = next
  })
  testActionSet('QuotaAccount.isLoading', () => loadingAccounts.value)
  testActionSet('QuotaAccount.getProviders', () => providers.value)
  testActionSet('QuotaAccount.getAccounts', () => accounts.value)
  testActionSet('QuotaAccount.getSnapshots', () => snapshots.value)
  testActionSet('QuotaAccount.getAccountCount', () => accounts.value.length)
  testActionSet('QuotaAccount.openAddModal', () => {
    doAdd()
    return true
  })
  testActionSet('QuotaAccount.fillForm', (params: unknown) => {
    modalRef.value?.testFill(
      params as {
        provider?: string
        name?: string
        key?: string
        note?: string
      }
    )
    return true
  })
  testActionSet('QuotaAccount.submitForm', async () => {
    await modalRef.value?.testSubmit()
    return true
  })
  testActionSet('QuotaAccount.searchProviders', (params: unknown) => {
    const { keyword } = (params ?? {}) as { keyword?: string }
    return modalRef.value?.testSearch(keyword ?? '') ?? []
  })
  testActionSet('QuotaAccount.addAccount', async (params: unknown) => {
    const p = params as {
      provider: string
      name: string
      key: string
      note: string
    }
    const account = await main.Call('quota.addAccount', p)
    await loadAccounts()
    return account
  })
  testActionSet('QuotaAccount.updateAccount', async (params: unknown) => {
    const p = params as { id: number; name: string; key: string; note: string }
    const account = await main.Call('quota.updateAccount', p)
    await loadAccounts()
    return account
  })
  testActionSet('QuotaAccount.deleteAccount', async (params: unknown) => {
    const { id } = params as { id: number }
    await main.Call('quota.deleteAccount', { id })
    await loadAccounts()
    return true
  })
  testActionSet('QuotaAccount.queryAccount', async (params: unknown) => {
    const { id } = params as { id: number }
    return await main.Call('quota.queryAccount', { id })
  })
  testActionSet('QuotaAccount.queryAll', async () => {
    await doQueryAll()
    return snapshots.value
  })
  testActionSet('QuotaAccount.renderScreenshot', (params: unknown) => {
    const { id } = params as { id: number }
    const account = accounts.value.find((item) => item.id === id)
    if (!account) throw new Error('账号不存在')
    return renderCardScreenshot(account, snapshots.value[id], appStore.locale)
  })
  testActionSet('QuotaAccount.previewSnapshot', (params: unknown) => {
    const snap = params as QuotaSnapshot
    snapshots.value = { ...snapshots.value, [snap.accountId]: snap }
    return true
  })
  testActionSet('QuotaAccount.openDetail', (params: unknown) => {
    const { id } = params as { id: number }
    const account = accounts.value.find((item) => item.id === id)
    if (!account) throw new Error('账号不存在')
    doDetail(account)
    return true
  })
  testActionSet('QuotaAccount.closeDetail', () => {
    showDetail.value = false
    return true
  })
  testActionSet('QuotaAccount.getDetailState', () => ({
    open: showDetail.value,
    accountId: detailAccount.value?.id ?? 0,
    windowCount: document.querySelectorAll('.quota-detail .detail-window').length,
    balanceCount: document.querySelectorAll('.quota-detail .detail-balance-row').length,
    statCount: document.querySelectorAll('.quota-detail .detail-stat-row').length,
  }))
  testActionSet('QuotaAccount.cardDimensions', () => {
    return Array.from(document.querySelectorAll('.quota-card')).map((card) => {
      const element = card as HTMLElement
      const details = element.querySelector('.card-details') as HTMLElement | null
      return {
        accountId: Number(element.dataset.accountId),
        top: element.getBoundingClientRect().top,
        height: element.getBoundingClientRect().height,
        scrollHeight: element.scrollHeight,
        detailsHeight: details?.getBoundingClientRect().height || 0,
        detailsScrollHeight: details?.scrollHeight || 0,
      }
    })
  })
  testActionSet('QuotaAccount.getThemeStyles', () => {
    const background = (selector: string) => {
      const element = document.querySelector(selector)
      return element ? getComputedStyle(element).backgroundColor : ''
    }
    return {
      dark: appStore.isDark,
      view: background('.quota-view'),
      header: background('.quota-view .view-header'),
      card: background('.quota-card'),
      sidebar: background('.app-sidebar'),
    }
  })
  testActionSet('QuotaAccount.setTheme', (params: unknown) => appStore.setTheme((params as { dark: boolean }).dark))

  // ── 截图 prepare / cleanup（截取前准备、截图后清理）────────────────────
  const prepareAccountPage = async () => {
    if (router.currentRoute.value.path !== '/quota/account') {
      await router.push('/quota/account')
    }
    if (providers.value.length === 0) await loadProviders()
    if (accounts.value.length === 0) await loadAccounts()
    if (Object.keys(snapshots.value).length === 0) await doQueryAll()
    await nextTick()
  }
  testActionSet('account.prepare', async () => {
    await prepareAccountPage()
    return true
  })
  testActionSet('account.cleanup', async () => {
    await nextTick()
    return true
  })
  testActionSet('account-add.prepare', async () => {
    await prepareAccountPage()
    doAdd()
    await nextTick()
    return true
  })
  testActionSet('account-add.cleanup', async () => {
    showModal.value = false
    await nextTick()
    return true
  })
  testActionSet('account-card.prepare', async () => {
    await prepareAccountPage()
    return true
  })
  testActionSet('account-card.cleanup', async () => {
    await nextTick()
    return true
  })
  testActionSet('account-dark.prepare', async () => {
    await prepareAccountPage()
    appStore.setTheme(true)
    await nextTick()
    return true
  })
  testActionSet('account-dark.cleanup', async () => {
    appStore.setTheme(false)
    await nextTick()
    return true
  })
})

onUnmounted(() => {
  EventsOff('quota:snapshots')
  testActionUnset([
    'QuotaAccount.isLoading',
    'QuotaAccount.getProviders',
    'QuotaAccount.getAccounts',
    'QuotaAccount.getSnapshots',
    'QuotaAccount.getAccountCount',
    'QuotaAccount.openAddModal',
    'QuotaAccount.fillForm',
    'QuotaAccount.submitForm',
    'QuotaAccount.searchProviders',
    'QuotaAccount.addAccount',
    'QuotaAccount.updateAccount',
    'QuotaAccount.deleteAccount',
    'QuotaAccount.queryAccount',
    'QuotaAccount.queryAll',
    'QuotaAccount.renderScreenshot',
    'QuotaAccount.previewSnapshot',
    'QuotaAccount.openDetail',
    'QuotaAccount.closeDetail',
    'QuotaAccount.getDetailState',
    'QuotaAccount.cardDimensions',
    'QuotaAccount.getThemeStyles',
    'QuotaAccount.setTheme',
    'account.prepare',
    'account.cleanup',
    'account-add.prepare',
    'account-add.cleanup',
    'account-card.prepare',
    'account-card.cleanup',
    'account-dark.prepare',
    'account-dark.cleanup',
  ])
})
</script>

<style scoped>
.quota-view {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f2f2f7;
  color: #1c1c1e;
  overflow: hidden;
}

:global(.dark .quota-view) {
  background: var(--token-surface);
  color: #f9fafb;
}
:global(.dark .empty-title) {
  color: #f9fafb;
}
:global(.dark .empty-desc) {
  color: #9ca3af;
}
:global(.dark .view-header) {
  background: var(--token-panel);
  border-bottom-color: #374151;
}

.view-header {
  flex-shrink: 0;
  padding: 10px 16px;
  background: #fff;
  border-bottom: 1px solid #e5e5ea;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  text-align: center;
}

.empty-icon {
  color: #d1d5db;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  color: #374151;
  margin: 0;
}

.empty-desc {
  font-size: 13px;
  color: #9ca3af;
  margin: 0;
  line-height: 1.5;
  max-width: 320px;
}

.account-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  grid-auto-rows: max-content;
  gap: 12px;
  align-content: start;
  align-items: stretch;
}

@media (min-width: 768px) {
  .account-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .account-list {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>

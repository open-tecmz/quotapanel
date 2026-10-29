<template>
  <div class="quota-view">
    <div class="view-header">
      <PageHeader :title="t('quota.nav.account')">
        <template #icon>
          <Wallet :size="22" class="text-primary-500" aria-hidden="true" />
        </template>
        <template #extra>
          <a-button :disabled="accounts.length === 0" @click="doQueryAll">
            <div class="inline-flex items-center gap-1">
              <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': queryingAll }" aria-hidden="true" />
              {{ t('quota.account.refresh') }}
            </div>
          </a-button>
          <a-button type="primary" @click="doAdd">
            <div class="inline-flex items-center gap-1">
              <Plus class="w-4 h-4" aria-hidden="true" />
              {{ t('quota.account.add') }}
            </div>
          </a-button>
          <a-dropdown :trigger="['click']" placement="bottomRight">
            <a-button
              :disabled="accounts.length === 0"
              :title="t('quota.account.moreActions')"
              :aria-label="t('quota.account.moreActions')"
            >
              <MoreHorizontal class="w-4 h-4" aria-hidden="true" />
            </a-button>
            <template #overlay>
              <a-menu @click="onHeaderAction">
                <a-menu-item key="share">
                  <span class="inline-flex items-center gap-2">
                    <ImageDown class="w-4 h-4" aria-hidden="true" />
                    {{ t('quota.account.shareAll') }}
                  </span>
                </a-menu-item>
                <a-menu-item key="copy">
                  <span class="inline-flex items-center gap-2">
                    <Copy class="w-4 h-4" aria-hidden="true" />
                    {{ t('quota.account.copyShare') }}
                  </span>
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
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

    <!-- 卡片列表（按住卡片左上角手柄可拖拽排序） -->
    <div v-else class="account-list" @dragover.prevent @drop.prevent="onCardDrop">
      <QuotaAccountCard
        v-for="account in accounts"
        :key="account.id"
        :account="account"
        :snapshot="snapshots[account.id]"
        :loading="queryingIds.has(account.id)"
        :browser-mode="providers.some((p) => p.id === account.provider && p.mode === 'browser')"
        :dragging="draggingId === account.id"
        :drop-side="dropSideOf(account)"
        @refresh="doQueryAccount(account.id)"
        @edit="doEdit(account)"
        @delete="doDelete(account)"
        @copy="doCopy(account)"
        @screenshot="doScreenshot(account)"
        @login="doBeginLogin(account.id)"
        @verify="doCompleteLogin(account.id)"
        @detail="doDetail(account)"
        @drag-start="onCardDragStart(account)"
        @drag-over="onCardDragOver(account, $event)"
        @drag-end="onCardDragEnd"
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

    <SharePreviewModal
      v-model:open="shareOpen"
      :png-base64="sharePng"
      :saving="shareSaving"
      :title="t('quota.account.sharePreviewTitle')"
      :hint="t('quota.account.sharePreviewHint')"
      @save="doSaveShare"
    />
  </div>
</template>

<script setup lang="ts">
import { message, Modal } from 'ant-design-vue'
import { Copy, ImageDown, MoreHorizontal, Plus, RefreshCw, Wallet } from 'lucide-vue-next'
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
import SharePreviewModal from './QuotaAccount/SharePreviewModal.vue'
import { renderCardScreenshot, renderShareImage } from '../utils/cardScreenshot'

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

const shareOpen = ref(false)
const sharePng = ref('')
const shareSaving = ref(false)
const website = ref('')

// ── 卡片拖拽排序 ──────────────────────────────────────────────
// draggingId：正在拖拽的账号；dropTarget：松手后的落位点；
// persistedOrder：已持久化的顺序（用于跳过无变化的重排请求）。
const draggingId = ref<number | null>(null)
const dropTarget = ref<{ id: number; after: boolean } | null>(null)
const persistedOrder = ref('')
let dragEndedAt = 0

function orderKey(list: QuotaAccount[]): string {
  return list.map((item) => item.id).join(',')
}

// moveCardTo 把账号 id 移动到目标下标（供拖拽落位与测试复用）。
function moveCardTo(id: number, toIndex: number): boolean {
  const fromIndex = accounts.value.findIndex((item) => item.id === id)
  if (fromIndex < 0) return false
  const clamped = Math.max(0, Math.min(toIndex, accounts.value.length - 1))
  if (fromIndex === clamped) return false
  const next = accounts.value.slice()
  const [moved] = next.splice(fromIndex, 1)
  next.splice(clamped, 0, moved)
  accounts.value = next
  return true
}

// persistOrder 把当前顺序写入后端；顺序未变化则不请求。
async function persistOrder() {
  const key = orderKey(accounts.value)
  if (key === persistedOrder.value) return
  try {
    await main.Call('quota.reorderAccounts', { ids: accounts.value.map((item) => item.id) })
    persistedOrder.value = key
    message.success(t('quota.account.reordered'))
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
    await loadAccounts()
  }
}

function onCardDragStart(account: QuotaAccount) {
  draggingId.value = account.id
}

function onCardDragOver(account: QuotaAccount, after: boolean) {
  if (draggingId.value === null || draggingId.value === account.id) {
    dropTarget.value = null
    return
  }
  dropTarget.value = { id: account.id, after }
}

function onCardDragEnd() {
  draggingId.value = null
  dropTarget.value = null
  dragEndedAt = Date.now()
}

// onCardDrop 在松手时按落位点重排并持久化。
async function onCardDrop() {
  const fromId = draggingId.value
  const target = dropTarget.value
  if (fromId === null) return
  if (target && target.id !== fromId) {
    const fromIndex = accounts.value.findIndex((item) => item.id === fromId)
    let toIndex = accounts.value.findIndex((item) => item.id === target.id)
    if (fromIndex >= 0 && toIndex >= 0) {
      if (target.after) toIndex += 1
      if (fromIndex < toIndex) toIndex -= 1
      moveCardTo(fromId, toIndex)
    }
  }
  onCardDragEnd()
  await persistOrder()
}

// dropSideOf 返回某张卡片应展示的落位指示方向。
function dropSideOf(account: QuotaAccount): 'before' | 'after' | null {
  const target = dropTarget.value
  if (!target || target.id !== account.id) return null
  return target.after ? 'after' : 'before'
}

async function loadAccounts() {
  loadingAccounts.value = true
  try {
    accounts.value = (await main.Call('quota.getAccounts')) || []
    persistedOrder.value = orderKey(accounts.value)
  } finally {
    loadingAccounts.value = false
  }
}

async function loadProviders() {
  providers.value = (await main.Call('quota.getProviders')) || []
}

async function doQueryAll() {
  if (accounts.value.length === 0 || queryingAll.value) return
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
  // 拖拽结束后浏览器可能补发一次 click，避免误开详情弹窗。
  if (Date.now() - dragEndedAt < 300) return
  detailAccount.value = account
  showDetail.value = true
}

// 菜单栏迷你面板点击某个账号：打开主窗口后定位到该账号详情。
async function doOpenAccountFromMini(id: number) {
  if (accounts.value.length === 0) await loadAccounts()
  const account = accounts.value.find((item) => item.id === id)
  if (account) doDetail(account)
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
    const pngBase64 = renderCardScreenshot(account, snapshots.value[account.id], appStore.locale, {
      website: website.value,
    })
    const saved = await main.Call('quota.saveImage', {
      title: t('quota.account.screenshot'),
      defaultName: `quota-card-${account.id}.png`,
      pngBase64,
    })
    if (saved) message.success(t('quota.account.screenshotSaved'))
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

// ── 分享图：列表长图 + 单卡截图共用同一套纸感绘制 ──────────────────
function buildShareImage(): string {
  return renderShareImage(accounts.value, snapshots.value, appStore.locale, {
    website: website.value,
  })
}

function shareFileName(): string {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `quotapanel-share-${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}.png`
}

function base64ToBlob(base64: string): Blob {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new Blob([bytes], { type: 'image/png' })
}

function onHeaderAction(info: { key: string | number }) {
  const key = String(info.key)
  if (key === 'share') void doShare()
  else if (key === 'copy') void doCopyShare()
}

async function doShare() {
  if (accounts.value.length === 0) return
  try {
    sharePng.value = buildShareImage()
    shareOpen.value = true
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function doCopyShare() {
  if (accounts.value.length === 0) return
  try {
    const pngBase64 = buildShareImage()
    const ClipboardItemCtor = (window as unknown as { ClipboardItem?: new (items: Record<string, Blob>) => unknown })
      .ClipboardItem
    if (!navigator.clipboard?.write || !ClipboardItemCtor) {
      throw new Error(t('quota.account.shareCopyUnsupported'))
    }
    const items = [new ClipboardItemCtor({ 'image/png': base64ToBlob(pngBase64) })]
    await (navigator.clipboard.write as unknown as (data: unknown[]) => Promise<void>)(items)
    message.success(t('quota.account.shareCopied'))
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  }
}

async function doSaveShare() {
  if (!sharePng.value || shareSaving.value) return
  shareSaving.value = true
  try {
    const saved = await main.Call('quota.saveImage', {
      title: t('quota.account.saveShareImage'),
      defaultName: shareFileName(),
      pngBase64: sharePng.value,
    })
    if (saved) {
      message.success(t('quota.account.shareSaved'))
      shareOpen.value = false
    }
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    shareSaving.value = false
  }
}

async function loadWebsite() {
  try {
    const config = await main.Call('setting.getAppConfig')
    website.value = String(config?.website || '')
  } catch {
    website.value = ''
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
  await loadWebsite()
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
  EventsOn('mini:openAccount', (id: number) => {
    doOpenAccountFromMini(id)
  })
  testActionSet('QuotaAccount.isLoading', () => loadingAccounts.value)
  testActionSet('QuotaAccount.isQueryingAll', () => queryingAll.value)
  testActionSet('QuotaAccount.getProviders', () => providers.value)
  testActionSet('QuotaAccount.getAccounts', () => accounts.value)
  testActionSet('QuotaAccount.getSnapshots', () => snapshots.value)
  testActionSet('QuotaAccount.getAccountCount', () => accounts.value.length)
  testActionSet('QuotaAccount.getCardOrder', () =>
    Array.from(document.querySelectorAll('.quota-card')).map((card) => Number((card as HTMLElement).dataset.accountId))
  )
  testActionSet('QuotaAccount.getCardHandleCount', () => document.querySelectorAll('.quota-card .card-handle').length)
  testActionSet('QuotaAccount.moveCard', async (params: unknown) => {
    const { id, toIndex } = params as { id: number; toIndex: number }
    moveCardTo(id, toIndex)
    await persistOrder()
    return accounts.value.map((item) => item.id)
  })
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
    return renderCardScreenshot(account, snapshots.value[id], appStore.locale, {
      website: website.value,
    })
  })
  testActionSet('QuotaAccount.renderShareImage', (params: unknown) => {
    const { includeNote } = (params ?? {}) as { includeNote?: boolean }
    return renderShareImage(accounts.value, snapshots.value, appStore.locale, {
      website: website.value,
      includeNote: includeNote === true,
    })
  })
  testActionSet('QuotaAccount.getShareState', () => ({
    open: shareOpen.value,
    hasImage: sharePng.value.length > 0,
    saving: shareSaving.value,
  }))
  testActionSet('QuotaAccount.openSharePreview', async () => {
    await doShare()
    return true
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
  testActionSet('QuotaAccount.openAccountFromMini', async (params: unknown) => {
    const { id } = params as { id: number }
    await doOpenAccountFromMini(id)
    return true
  })
  testActionSet('QuotaAccount.getDetailState', () => {
    const body = document.querySelector('.quota-detail') as HTMLElement | null
    const bodyStyle = body ? getComputedStyle(body) : null
    return {
      open: showDetail.value,
      accountId: detailAccount.value?.id ?? 0,
      windowCount: document.querySelectorAll('.quota-detail .detail-window').length,
      balanceCount: document.querySelectorAll('.quota-detail .detail-balance-row').length,
      statCount: document.querySelectorAll('.quota-detail .detail-stat-row').length,
      bodyMaxHeight: bodyStyle?.maxHeight ?? '',
      bodyScrollHeight: body?.scrollHeight ?? 0,
      bodyClientHeight: body?.clientHeight ?? 0,
    }
  })
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
  testActionSet('account-detail.prepare', async () => {
    await prepareAccountPage()
    const first = accounts.value[0]
    if (first) doDetail(first)
    await nextTick()
    return true
  })
  testActionSet('account-detail.cleanup', async () => {
    showDetail.value = false
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
  EventsOff('mini:openAccount')
  testActionUnset([
    'QuotaAccount.isLoading',
    'QuotaAccount.isQueryingAll',
    'QuotaAccount.getProviders',
    'QuotaAccount.getAccounts',
    'QuotaAccount.getSnapshots',
    'QuotaAccount.getAccountCount',
    'QuotaAccount.getCardOrder',
    'QuotaAccount.getCardHandleCount',
    'QuotaAccount.moveCard',
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
    'QuotaAccount.renderShareImage',
    'QuotaAccount.getShareState',
    'QuotaAccount.openSharePreview',
    'QuotaAccount.previewSnapshot',
    'QuotaAccount.openDetail',
    'QuotaAccount.closeDetail',
    'QuotaAccount.openAccountFromMini',
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
    'account-detail.prepare',
    'account-detail.cleanup',
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

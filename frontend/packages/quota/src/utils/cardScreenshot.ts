import { designTokens } from '@quotapanel/ui/src/theme/tokens'
import type { QuotaAccount, QuotaSnapshot } from '../api/main'
import { firstLine, formatDateTime, hasResetTime, statusColor, statusLabel, aggregateStatus } from './format'

type Locale = 'zh' | 'en'

export interface ShareImageOptions {
  /** 页脚展示的官网地址（来自 setting.getAppConfig.website）。 */
  website?: string
  /** 分享长图是否包含备注；默认不包含，避免把私人备注发到社交媒体。 */
  includeNote?: boolean
}

const DEFAULT_WEBSITE = 'open.tecmz.com/quotapanel'
const FONT = 'Inter, system-ui, -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif'

// 分享图使用固定的「纸感」浅色主题：社交媒体上不依赖用户当前的明暗主题。
const palette = {
  background: designTokens.color.neutral[50],
  dot: 'rgba(148, 163, 184, 0.35)',
  card: '#ffffff',
  border: designTokens.color.neutral[200],
  divider: designTokens.color.neutral[200],
  text: designTokens.color.neutral[900],
  muted: designTokens.color.neutral[500],
  soft: designTokens.color.neutral[400],
  primary: designTokens.color.primary,
  primaryHover: designTokens.color.primaryHover,
  primarySoft: '#f0f9f9',
  primaryBorder: 'rgba(1, 139, 141, 0.18)',
  track: designTokens.color.neutral[200],
  danger: designTokens.color.danger,
  dangerSoft: '#fef2f2',
  warning: '#f59e0b',
  warningText: designTokens.color.warningStrong,
  neutralSoft: '#f1f5f9',
}

function font(size: number, weight: string | number = 400): string {
  return `${weight} ${size}px ${FONT}`
}

function roundRectPath(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number
) {
  const r = Math.max(0, Math.min(radius, width / 2, height / 2))
  ctx.beginPath()
  ctx.moveTo(x + r, y)
  ctx.arcTo(x + width, y, x + width, y + height, r)
  ctx.arcTo(x + width, y + height, x, y + height, r)
  ctx.arcTo(x, y + height, x, y, r)
  ctx.arcTo(x, y, x + width, y, r)
  ctx.closePath()
}

function fillRound(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
  fill: string | CanvasGradient
) {
  if (width <= 0 || height <= 0) return
  roundRectPath(ctx, x, y, width, height, radius)
  ctx.fillStyle = fill
  ctx.fill()
}

function strokeRound(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
  stroke: string
) {
  if (width <= 0 || height <= 0) return
  roundRectPath(ctx, x, y, width, height, radius)
  ctx.strokeStyle = stroke
  ctx.lineWidth = 1
  ctx.stroke()
}

function ellipsis(ctx: CanvasRenderingContext2D, value: string, maxWidth: number): string {
  if (maxWidth <= 0) return ''
  if (ctx.measureText(value).width <= maxWidth) return value
  let shown = value
  while (shown.length > 1 && ctx.measureText(`${shown}…`).width > maxWidth) {
    shown = shown.slice(0, -1)
  }
  return `${shown}…`
}

function text(
  ctx: CanvasRenderingContext2D,
  value: string,
  x: number,
  y: number,
  maxWidth: number,
  size: number,
  color: string,
  weight: string | number = 400,
  align: CanvasTextAlign = 'left'
) {
  ctx.font = font(size, weight)
  ctx.fillStyle = color
  ctx.textBaseline = 'middle'
  ctx.textAlign = align
  ctx.fillText(ellipsis(ctx, value ?? '', maxWidth), x, y)
  ctx.textAlign = 'left'
}

function line(ctx: CanvasRenderingContext2D, x1: number, y: number, x2: number) {
  ctx.strokeStyle = palette.divider
  ctx.lineWidth = 1
  ctx.beginPath()
  ctx.moveTo(x1, y + 0.5)
  ctx.lineTo(x2, y + 0.5)
  ctx.stroke()
}

function drawPaperBackground(ctx: CanvasRenderingContext2D, width: number, height: number) {
  ctx.fillStyle = palette.background
  ctx.fillRect(0, 0, width, height)
  ctx.fillStyle = palette.dot
  for (let y = 6; y < height; y += 12) {
    for (let x = 6; x < width; x += 12) {
      ctx.beginPath()
      ctx.arc(x, y, 0.6, 0, Math.PI * 2)
      ctx.fill()
    }
  }
}

function drawBrandMark(ctx: CanvasRenderingContext2D, x: number, y: number, size: number) {
  const gradient = ctx.createLinearGradient(x, y, x + size, y + size)
  gradient.addColorStop(0, designTokens.color.primaryDark)
  gradient.addColorStop(0.72, designTokens.color.primary)
  fillRound(ctx, x, y, size, size, size * 0.3, gradient)
  text(ctx, 'Q', x + size / 2, y + size / 2 + 0.5, size, size * 0.56, '#ffffff', 800, 'center')
}

function displayWebsite(website?: string): string {
  const value = (website || DEFAULT_WEBSITE).trim()
  if (!value) return DEFAULT_WEBSITE
  return value.replace(/^https?:\/\//i, '').replace(/\/+$/, '')
}

function statusInfo(status: string | undefined, locale: Locale) {
  const key =
    status === 'ok' || status === 'active'
      ? 'ok'
      : status === 'warning'
        ? 'warning'
        : status === 'exceeded'
          ? 'exceeded'
          : 'unknown'
  const labels =
    locale === 'en'
      ? { ok: 'Normal', warning: 'Nearly exhausted', exceeded: 'Exceeded', unknown: 'Unknown' }
      : {
          ok: statusLabel('ok'),
          warning: statusLabel('warning'),
          exceeded: statusLabel('exceeded'),
          unknown: statusLabel(undefined),
        }
  const colors = {
    ok: { fg: palette.primaryHover, bg: 'rgba(1, 139, 141, 0.10)' },
    warning: { fg: palette.warningText, bg: 'rgba(245, 158, 11, 0.16)' },
    exceeded: { fg: palette.danger, bg: 'rgba(239, 68, 68, 0.11)' },
    unknown: { fg: palette.soft, bg: palette.neutralSoft },
  }
  return { label: labels[key], ...colors[key] }
}

function drawStatusPill(
  ctx: CanvasRenderingContext2D,
  label: string,
  color: string,
  background: string,
  right: number,
  centerY: number,
  height = 20,
  padding = 8
) {
  ctx.font = font(9, 600)
  const width = ctx.measureText(label).width + padding * 2
  fillRound(ctx, right - width, centerY - height / 2, width, height, height / 2, background)
  text(ctx, label, right - width / 2, centerY, width - padding, 9, color, 600, 'center')
  return width
}

function latestUpdate(snapshots: Record<number, QuotaSnapshot>): string {
  let latest = 0
  for (const snapshot of Object.values(snapshots)) {
    const time = new Date(snapshot.updatedAt || '').getTime()
    if (!Number.isNaN(time) && time > latest) latest = time
  }
  return formatDateTime(latest ? new Date(latest).toISOString() : new Date().toISOString())
}

interface StatusCounts {
  total: number
  ok: number
  warning: number
  exceeded: number
  unknown: number
}

function countStatuses(accounts: QuotaAccount[], snapshots: Record<number, QuotaSnapshot>): StatusCounts {
  const counts: StatusCounts = { total: accounts.length, ok: 0, warning: 0, exceeded: 0, unknown: 0 }
  for (const account of accounts) {
    const snapshot = snapshots[account.id]
    if (!snapshot) {
      counts.unknown += 1
      continue
    }
    if (snapshot.error) {
      counts.exceeded += 1
      continue
    }
    const status = aggregateStatus(snapshot.status, snapshot.windows)
    if (status === 'exceeded') counts.exceeded += 1
    else if (status === 'warning') counts.warning += 1
    else if (status === 'ok' || status === 'active') counts.ok += 1
    else counts.unknown += 1
  }
  return counts
}

function chipWidth(ctx: CanvasRenderingContext2D, label: string, count: number): number {
  ctx.font = font(11, 500)
  const labelWidth = ctx.measureText(label).width
  ctx.font = font(11, 650)
  const countWidth = ctx.measureText(String(count)).width
  return 34 + labelWidth + countWidth
}

function drawChip(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  label: string,
  count: number,
  dotColor: string
): number {
  const width = chipWidth(ctx, label, count)
  ctx.font = font(11, 650)
  const countWidth = ctx.measureText(String(count)).width
  fillRound(ctx, x, y, width, 24, 12, palette.card)
  strokeRound(ctx, x, y, width, 24, 12, palette.border)
  ctx.beginPath()
  ctx.arc(x + 13, y + 12, 2.5, 0, Math.PI * 2)
  ctx.fillStyle = dotColor
  ctx.fill()
  text(ctx, label, x + 20, y + 12, width - 20 - countWidth - 14, 11, palette.muted, 500)
  text(ctx, String(count), x + width - 8, y + 12, countWidth + 2, 11, palette.text, 650, 'right')
  return width
}

function drawHeader(
  ctx: CanvasRenderingContext2D,
  width: number,
  accounts: QuotaAccount[],
  snapshots: Record<number, QuotaSnapshot>,
  locale: Locale
): number {
  drawBrandMark(ctx, 32, 30, 28)
  text(ctx, 'QuotaPanel', 68, 44, 200, 14, palette.text, 700)
  text(ctx, formatDateTime(new Date().toISOString()), width - 32, 44, 170, 10, palette.soft, 400, 'right')

  const title = locale === 'en' ? 'My AI subscriptions' : '我的 AI 订阅额度'
  text(ctx, title, 32, 100, width - 64, 26, palette.text, 750)

  const subtitle =
    locale === 'en'
      ? `${accounts.length} subscriptions · Quota updated ${latestUpdate(snapshots)}`
      : `${accounts.length} 个订阅 · 额度更新于 ${latestUpdate(snapshots)}`
  text(ctx, subtitle, 32, 126, width - 64, 11, palette.muted)

  const counts = countStatuses(accounts, snapshots)
  let x = 32
  const y = 146
  const chips: Array<[string, number, string]> = [
    [locale === 'en' ? 'Subscriptions' : '订阅', counts.total, palette.soft],
    [locale === 'en' ? 'Normal' : '正常', counts.ok, palette.primary],
    [locale === 'en' ? 'Nearly exhausted' : '即将用尽', counts.warning, palette.warning],
    [locale === 'en' ? 'Exceeded' : '已超限', counts.exceeded, palette.danger],
  ]
  if (counts.unknown > 0) {
    chips.push([locale === 'en' ? 'Not checked' : '未查询', counts.unknown, palette.soft])
  }
  for (const [label, count, color] of chips) {
    const widthOfChip = drawChip(ctx, x, y, label, count, color)
    x += widthOfChip + 6
  }
  return 186
}

function rowCount(snapshot?: QuotaSnapshot): number {
  if (!snapshot || snapshot.error) return 1
  const windows = (snapshot.windows || []).slice(0, 3)
  if (windows.length > 0) return windows.length
  const stats = [...(snapshot.balances || []), ...(snapshot.stats || [])].slice(0, 3)
  return Math.max(1, stats.length)
}

function longCardHeight(account: QuotaAccount, snapshot: QuotaSnapshot | undefined, includeNote: boolean): number {
  const noteHeight = includeNote && account.note ? 14 : 0
  return 48 + noteHeight + rowCount(snapshot) * 34 + 12
}

function drawLongCard(
  ctx: CanvasRenderingContext2D,
  account: QuotaAccount,
  snapshot: QuotaSnapshot | undefined,
  x: number,
  y: number,
  width: number,
  height: number,
  locale: Locale,
  includeNote: boolean
) {
  fillRound(ctx, x, y, width, height, 11, palette.card)
  strokeRound(ctx, x, y, width, height, 11, palette.border)

  const provider = account.providerName?.trim() || ''
  const plan = snapshot?.plan?.trim() || ''
  const subtitle = !plan || plan === provider ? provider : `${provider} · ${plan}`
  const noteHeight = includeNote && account.note ? 14 : 0

  text(ctx, account.name, x + 12, y + 20, width - 100, 13, palette.text, 650)
  text(ctx, subtitle, x + 12, y + 36, width - 100, 10, palette.muted)
  if (noteHeight > 0) {
    text(ctx, firstLine(account.note), x + 12, y + 50, width - 100, 9, palette.soft)
  }

  const status = snapshot
    ? statusInfo(snapshot.error ? 'exceeded' : aggregateStatus(snapshot.status, snapshot.windows), locale)
    : { label: locale === 'en' ? 'Not checked' : '未查询', fg: palette.soft, bg: palette.neutralSoft }
  drawStatusPill(ctx, status.label, status.fg, status.bg, x + width - 12, y + 20)

  let rowY = y + 48 + noteHeight
  if (!snapshot) {
    text(
      ctx,
      locale === 'en' ? 'Refresh to check quota' : '点击刷新查询额度',
      x + width / 2,
      rowY + 10,
      width - 24,
      10,
      palette.soft,
      400,
      'center'
    )
    return
  }

  if (snapshot.error) {
    fillRound(ctx, x + 12, rowY - 10, width - 24, 34, 8, palette.dangerSoft)
    text(ctx, snapshot.error, x + 20, rowY + 7, width - 40, 9.5, palette.danger)
    return
  }

  const windows = (snapshot.windows || []).slice(0, 3)
  if (windows.length > 0) {
    for (const window of windows) {
      text(ctx, window.label, x + 12, rowY, width - 90, 10, palette.muted)
      text(
        ctx,
        window.detail || `${Math.round(window.percent)}%`,
        x + width - 12,
        rowY,
        80,
        10,
        palette.text,
        600,
        'right'
      )
      const barWidth = width - 24
      fillRound(ctx, x + 12, rowY + 7, barWidth, 4, 2, palette.track)
      const percent = Math.max(0, Math.min(100, window.percent))
      if (percent > 0) {
        fillRound(ctx, x + 12, rowY + 7, (barWidth * percent) / 100, 4, 2, statusColor(window.status))
      }
      if (hasResetTime(window.resetAt)) {
        text(ctx, window.resetAt, x + 12, rowY + 18, width - 24, 8.5, palette.soft)
      }
      rowY += 34
    }
    return
  }

  const stats = [...(snapshot.balances || []), ...(snapshot.stats || [])].slice(0, 3)
  if (stats.length > 0) {
    for (const stat of stats) {
      text(ctx, stat.label, x + 12, rowY, width - 120, 10, palette.muted)
      text(ctx, stat.value, x + width - 12, rowY, 110, 10, stat.muted ? palette.soft : palette.text, 600, 'right')
      rowY += 34
    }
    return
  }

  text(
    ctx,
    locale === 'en' ? 'No quota data' : '暂无额度信息',
    x + width / 2,
    rowY + 10,
    width - 24,
    10,
    palette.soft,
    400,
    'center'
  )
}

function drawFooter(ctx: CanvasRenderingContext2D, width: number, footerTop: number, website: string, locale: Locale) {
  line(ctx, 32, footerTop, width - 32)
  drawBrandMark(ctx, 32, footerTop + 18, 24)
  text(
    ctx,
    locale === 'en' ? 'Generated with QuotaPanel' : '使用 QuotaPanel 生成',
    64,
    footerTop + 26,
    width - 96,
    12,
    palette.text,
    650
  )
  text(ctx, website, 64, footerTop + 42, width - 96, 9.5, palette.soft)
  text(
    ctx,
    locale === 'en' ? 'Free · Open source · Multi-provider quota tracker' : '免费 · 开源 · 多平台订阅额度管理',
    width - 32,
    footerTop + 34,
    320,
    9,
    palette.soft,
    400,
    'right'
  )
}

// 把当前列表绘制成一张适合社交媒体分享的长图（2x PNG，base64）。
export function renderShareImage(
  accounts: QuotaAccount[],
  snapshots: Record<number, QuotaSnapshot>,
  locale: Locale = 'zh',
  options: ShareImageOptions = {}
): string {
  const width = 800
  const padding = 32
  const gap = 12
  const includeNote = options.includeNote === true
  const website = displayWebsite(options.website)

  // 只有一个账号时用单列宽卡居中，避免两侧留白过多。
  const columnWidth = accounts.length === 1 ? 520 : (width - padding * 2 - gap) / 2
  const rows: QuotaAccount[][] = []
  for (let index = 0; index < accounts.length; index += 2) {
    rows.push(accounts.slice(index, index + 2))
  }

  const rowHeights = rows.map((row) =>
    Math.max(...row.map((account) => longCardHeight(account, snapshots[account.id], includeNote)))
  )
  const gridTop = 186
  const gridBottom = gridTop + rowHeights.reduce((sum, height) => sum + height + gap, 0)
  const footerTop = accounts.length > 0 ? gridBottom + 18 : 250
  const height = footerTop + 64

  const scale = 2
  const canvas = document.createElement('canvas')
  canvas.width = width * scale
  canvas.height = height * scale
  const context = canvas.getContext('2d')
  if (!context) throw new Error('无法创建分享图画布')
  const ctx: CanvasRenderingContext2D = context
  ctx.scale(scale, scale)

  drawPaperBackground(ctx, width, height)
  drawHeader(ctx, width, accounts, snapshots, locale)

  if (accounts.length === 0) {
    text(
      ctx,
      locale === 'en' ? 'No subscriptions yet' : '暂无订阅',
      width / 2,
      220,
      width - 64,
      14,
      palette.soft,
      500,
      'center'
    )
  }

  let y = gridTop
  rows.forEach((row, rowIndex) => {
    const rowHeight = rowHeights[rowIndex]
    const rowWidth = row.length * columnWidth + (row.length - 1) * gap
    let x = row.length === 1 ? (width - rowWidth) / 2 : padding
    for (const account of row) {
      drawLongCard(ctx, account, snapshots[account.id], x, y, columnWidth, rowHeight, locale, includeNote)
      x += columnWidth + gap
    }
    y += rowHeight + gap
  })

  drawFooter(ctx, width, footerTop, website, locale)
  return canvas.toDataURL('image/png').split(',')[1]
}

// 单卡截图：卡片四周留白 + 同款纸感背景 + 品牌页脚。
export function renderCardScreenshot(
  account: QuotaAccount,
  snapshot?: QuotaSnapshot,
  locale: Locale = 'zh',
  options: ShareImageOptions = {}
): string {
  const width = 600
  const padding = 24
  const cardWidth = width - padding * 2
  const windows = snapshot?.windows || []
  const stats = [...(snapshot?.balances || []), ...(snapshot?.stats || [])]
  const contentHeight = snapshot?.error
    ? 42
    : snapshot
      ? windows.length * 39 + stats.length * 20 + (snapshot.summary ? 18 : 0)
      : 24
  const cardFooterTop = Math.max(214, 124 + contentHeight + 12)
  const cardHeight = cardFooterTop + 42
  const footerTop = padding + cardHeight + 18
  const height = footerTop + 52
  const website = displayWebsite(options.website)

  const scale = 2
  const canvas = document.createElement('canvas')
  canvas.width = width * scale
  canvas.height = height * scale
  const context = canvas.getContext('2d')
  if (!context) throw new Error('无法创建截图画布')
  const ctx: CanvasRenderingContext2D = context
  ctx.scale(scale, scale)

  drawPaperBackground(ctx, width, height)

  ctx.save()
  ctx.translate(padding, padding)
  fillRound(ctx, 0.5, 0.5, cardWidth - 1, cardHeight - 1, 14, palette.card)
  strokeRound(ctx, 0.5, 0.5, cardWidth - 1, cardHeight - 1, 14, palette.border)

  fillRound(ctx, 16, 16, 38, 38, 10, palette.primarySoft)
  strokeRound(ctx, 16, 16, 38, 38, 10, palette.primaryBorder)
  text(ctx, 'Q', 35, 35.5, 24, 18, palette.primaryHover, 800, 'center')

  text(ctx, account.name, 66, 29, cardWidth - 190, 16, palette.text, 600)
  const provider = account.providerName?.trim() || ''
  const plan = snapshot?.plan?.trim() || ''
  const subtitle = !plan || plan === provider ? provider : `${provider} · ${plan}`
  text(ctx, subtitle, 66, 48, cardWidth - 190, 11, palette.soft)

  const snapStatus = aggregateStatus(snapshot?.error ? 'exceeded' : snapshot?.status, snapshot?.windows || [])
  const status = snapshot
    ? statusInfo(snapshot.error ? 'exceeded' : snapStatus, locale)
    : { label: locale === 'en' ? 'Not checked' : '未查询', fg: palette.soft, bg: palette.neutralSoft }
  fillRound(ctx, cardWidth - 104, 22, 86, 26, 13, status.bg)
  text(ctx, status.label, cardWidth - 61, 35, 76, 12, status.fg, 500, 'center')

  if (account.note) text(ctx, firstLine(account.note), 66, 71, cardWidth - 82, 11, palette.muted)
  text(ctx, account.keyMasked, 66, account.note ? 89 : 76, cardWidth - 82, 11, palette.soft)
  line(ctx, 16, 106, cardWidth - 16)

  let y = 124
  if (snapshot?.error) {
    fillRound(ctx, 16, y - 10, cardWidth - 32, 42, 8, palette.dangerSoft)
    text(ctx, snapshot.error, 26, y + 10, cardWidth - 52, 12, palette.danger)
  } else if (snapshot) {
    for (const window of windows) {
      text(ctx, window.label, 16, y, 250, 11, palette.muted)
      text(
        ctx,
        window.detail || `${Math.round(window.percent)}%`,
        cardWidth - 16,
        y,
        180,
        11,
        palette.muted,
        400,
        'right'
      )
      const barWidth = cardWidth - 32
      fillRound(ctx, 16, y + 11, barWidth, 6, 3, palette.track)
      const percent = Math.max(0, Math.min(100, window.percent))
      if (percent > 0) {
        fillRound(ctx, 16, y + 11, (barWidth * percent) / 100, 6, 3, statusColor(window.status))
      }
      if (hasResetTime(window.resetAt)) {
        const resetLabel = locale === 'en' ? 'Resets' : '重置时间'
        text(ctx, `${resetLabel}：${window.resetAt}`, 16, y + 27, barWidth, 10, palette.soft)
      }
      y += 39
    }
    for (const stat of stats) {
      text(ctx, stat.label, 16, y, 240, 11, palette.muted)
      text(ctx, stat.value, cardWidth - 16, y, 244, 11, stat.muted ? palette.soft : palette.text, 500, 'right')
      y += 20
    }
    if (snapshot.summary) {
      text(ctx, snapshot.summary, 16, y + 2, cardWidth - 32, 11, palette.primaryHover)
    }
  } else {
    text(
      ctx,
      locale === 'en' ? 'Refresh to check quota' : '点击刷新查询额度',
      16,
      y + 10,
      cardWidth - 32,
      12,
      palette.soft
    )
  }

  line(ctx, 16, cardFooterTop, cardWidth - 16)
  const time = snapshot?.updatedAt
    ? `${locale === 'en' ? 'Quota updated' : '额度更新于'} ${formatDateTime(snapshot.updatedAt)}`
    : `${locale === 'en' ? 'Account updated' : '账号更新于'} ${formatDateTime(account.updatedAt)}`
  text(ctx, time, 16, cardFooterTop + 20, cardWidth - 32, 11, palette.soft)
  ctx.restore()

  line(ctx, padding, footerTop, width - padding)
  drawBrandMark(ctx, padding, footerTop + 14, 20)
  text(
    ctx,
    locale === 'en' ? 'Generated with QuotaPanel' : '使用 QuotaPanel 生成',
    padding + 28,
    footerTop + 19,
    width - padding * 2 - 28,
    11,
    palette.text,
    600
  )
  text(ctx, website, padding + 28, footerTop + 34, width - padding * 2 - 28, 9, palette.soft)

  return canvas.toDataURL('image/png').split(',')[1]
}

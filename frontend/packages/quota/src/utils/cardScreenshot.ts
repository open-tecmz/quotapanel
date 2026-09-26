import { designTokens } from '@quotapanel/ui/src/theme/tokens'
import type { QuotaAccount, QuotaSnapshot } from '../api/main'
import { formatDateTime, statusColor, statusLabel } from './format'

// 用 Canvas 绘制完整卡片内容，避免 WebKit 对 HTML foreignObject 截图的限制。
export function renderCardScreenshot(
  account: QuotaAccount,
  snapshot?: QuotaSnapshot,
  locale: 'zh' | 'en' = 'zh'
): string {
  const width = 560
  const stats = [...(snapshot?.balances || []), ...(snapshot?.stats || [])]
  const contentHeight = snapshot?.error
    ? 42
    : snapshot
      ? (snapshot.plan ? 20 : 0) +
        (snapshot.windows?.length || 0) * 39 +
        stats.length * 20 +
        (snapshot.summary ? 18 : 0)
      : 24
  const footerTop = Math.max(278, 124 + contentHeight + 12)
  const height = footerTop + 42
  const scale = 2
  const canvas = document.createElement('canvas')
  canvas.width = width * scale
  canvas.height = height * scale
  const context = canvas.getContext('2d')
  if (!context) throw new Error('无法创建截图画布')
  const ctx: CanvasRenderingContext2D = context
  ctx.scale(scale, scale)

  const color = {
    dark: designTokens.color.surfaceDark,
    gray: '#6b7280',
    light: '#9ca3af',
    border: '#e5e7eb',
    green: designTokens.color.primaryHover,
  }
  function box(x: number, y: number, w: number, h: number, radius: number, fill: string, stroke?: string) {
    ctx.beginPath()
    ctx.roundRect(x, y, w, h, radius)
    ctx.fillStyle = fill
    ctx.fill()
    if (stroke) {
      ctx.strokeStyle = stroke
      ctx.stroke()
    }
  }
  function line(text: string, x: number, y: number, maxWidth: number, size = 12, fill = color.gray, weight = 'normal') {
    ctx.font = `${weight} ${size}px -apple-system, BlinkMacSystemFont, sans-serif`
    ctx.fillStyle = fill
    ctx.textBaseline = 'middle'
    let shown = text
    while (shown.length > 1 && ctx.measureText(shown).width > maxWidth) shown = shown.slice(0, -1)
    if (shown !== text) shown = shown.slice(0, -1) + '…'
    ctx.fillText(shown, x, y)
  }

  box(0.5, 0.5, width - 1, height - 1, 14, '#ffffff', color.border)
  box(16, 16, 38, 38, 10, '#f0f9f9', '#dbf5f5')
  line('Q', 28, 35, 20, 18, color.green, 'bold')
  line(account.name, 66, 29, 340, 16, color.dark, '600')
  line(account.providerName, 66, 48, 340, 11, color.light)
  const label = snapshot
    ? locale === 'en'
      ? (
          { ok: 'Normal', active: 'Normal', warning: 'Nearly exhausted', exceeded: 'Exceeded' } as Record<
            string,
            string
          >
        )[snapshot.error ? 'exceeded' : snapshot.status] || 'Unknown'
      : statusLabel(snapshot.error ? 'exceeded' : snapshot.status)
    : locale === 'en'
      ? 'Not checked'
      : '未查询'
  box(456, 22, 86, 26, 13, snapshot?.error ? '#fef2f2' : '#f0f9f9')
  line(label, 478, 35, 60, 12, snapshot?.error ? '#dc2626' : color.green)

  if (account.note) line(account.note, 66, 71, 460, 11)
  line(account.keyMasked, 66, account.note ? 89 : 76, 460, 11, color.light)
  ctx.strokeStyle = '#f3f4f6'
  ctx.beginPath()
  ctx.moveTo(16, 106)
  ctx.lineTo(544, 106)
  ctx.stroke()

  let y = 124
  if (snapshot?.error) {
    box(16, y - 10, 528, 42, 8, '#fef2f2')
    line(snapshot.error, 26, y + 10, 504, 12, '#dc2626')
  } else if (snapshot) {
    if (snapshot.plan) {
      line(snapshot.plan, 16, y, 528, 12, color.gray, '600')
      y += 20
    }
    for (const window of snapshot.windows || []) {
      line(window.label, 16, y, 250, 11)
      line(window.detail || `${Math.round(window.percent)}%`, 362, y, 180, 11)
      box(16, y + 11, 528, 6, 3, '#e5e7eb')
      box(16, y + 11, (528 * Math.max(0, Math.min(100, window.percent))) / 100, 6, 3, statusColor(window.status))
      line(`${locale === 'en' ? 'Resets' : '重置时间'}：${window.resetAt || '-'}`, 16, y + 27, 528, 10, color.light)
      y += 39
    }
    for (const stat of stats) {
      line(stat.label, 16, y, 240, 11)
      line(stat.value, 300, y, 244, 11, stat.muted ? color.light : color.dark, '500')
      y += 20
    }
    if (snapshot.summary) line(snapshot.summary, 16, y + 2, 528, 11, color.green)
  } else {
    line(locale === 'en' ? 'Refresh to check quota' : '点击刷新查询额度', 16, y + 10, 528, 12, color.light)
  }

  ctx.beginPath()
  ctx.moveTo(16, footerTop)
  ctx.lineTo(544, footerTop)
  ctx.stroke()
  const time = snapshot?.updatedAt
    ? `${locale === 'en' ? 'Quota updated' : '额度更新于'} ${formatDateTime(snapshot.updatedAt)}`
    : `${locale === 'en' ? 'Account updated' : '账号更新于'} ${formatDateTime(account.updatedAt)}`
  line(time, 16, footerTop + 20, 528, 11, color.light)
  return canvas.toDataURL('image/png').split(',')[1]
}

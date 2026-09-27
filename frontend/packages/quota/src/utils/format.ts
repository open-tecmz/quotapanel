import { designTokens } from '@quotapanel/ui/src/theme/tokens'

// 将 ISO 时间字符串格式化为本地可读时间（YYYY-MM-DD HH:mm）。
export function formatDateTime(v: string | null | undefined): string {
  if (!v) return '-'
  const d = new Date(v)
  if (isNaN(d.getTime())) return v
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 按状态返回进度条颜色。
export function statusColor(status: string | undefined): string {
  switch (status) {
    case 'exceeded':
      return designTokens.color.danger
    case 'warning':
      return '#f59e0b'
    case 'ok':
      return designTokens.color.primary
    default:
      return designTokens.color.neutral[400]
  }
}

// 按状态返回中文状态文案。
export function statusLabel(status: string | undefined): string {
  switch (status) {
    case 'exceeded':
      return '已超限'
    case 'warning':
      return '即将用尽'
    case 'ok':
    case 'active':
      return '正常'
    default:
      return '未知'
  }
}

// 判断额度窗口的重置 / 周期时间是否有效（空值与占位符 '-' 视为无周期）。
export function hasResetTime(v: string | null | undefined): boolean {
  const s = (v ?? '').trim()
  return s !== '' && s !== '-'
}

// 按状态返回 Ant 标签颜色。
export function statusTagColor(status: string | undefined): string {
  switch (status) {
    case 'exceeded':
      return 'error'
    case 'warning':
      return 'warning'
    case 'ok':
    case 'active':
      return 'success'
    default:
      return 'default'
  }
}

// 按状态返回 i18n 文案 key（quota.account.statusXxx），供卡片与详情弹窗复用。
export function statusTextKey(
  status: string | undefined
): 'statusOk' | 'statusWarning' | 'statusExceeded' | 'statusUnknown' {
  switch (status) {
    case 'ok':
    case 'active':
      return 'statusOk'
    case 'warning':
      return 'statusWarning'
    case 'exceeded':
      return 'statusExceeded'
    default:
      return 'statusUnknown'
  }
}

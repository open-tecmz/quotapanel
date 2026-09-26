/**
 * 运行平台检测
 *
 * 通过 navigator 信息判断当前系统，用于标题栏按平台呈现不同样式：
 * - macOS：保留系统原生红绿灯，标题栏仅保留拖拽区
 * - Windows / Linux：无边框窗口，由前端自绘模拟系统的窗口按钮
 */

export type AppPlatform = 'mac' | 'windows' | 'linux'

const PLATFORM_LABELS: Record<AppPlatform, string> = {
  mac: 'macOS',
  windows: 'Windows',
  linux: 'Linux',
}

/**
 * 检测当前运行平台。基于 navigator.platform 与 userAgent 综合判断，
 * 兼容 WKWebView / WebView2 / WebKitGTK 的常见 UA 格式。
 */
export function detectPlatform(): AppPlatform {
  const source = `${navigator.platform ?? ''} ${navigator.userAgent ?? ''}`.toLowerCase()
  if (source.includes('mac') || source.includes('darwin')) return 'mac'
  if (source.includes('win')) return 'windows'
  return 'linux'
}

/**
 * 平台中文/英文展示名，用于调试与测试信息展示。
 */
export function platformLabel(platform: AppPlatform): string {
  return PLATFORM_LABELS[platform]
}

/**
 * 是否使用系统原生窗口控制按钮（红绿灯）。
 */
export function hasNativeWindowControls(platform: AppPlatform): boolean {
  return platform === 'mac'
}

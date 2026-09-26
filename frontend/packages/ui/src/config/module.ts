// 本文件是 frontend/packages/ui 中唯一包含 #if TYPE_XXX 条件编译注释的文件。
// 所有模块特定的条件编译集中在此处，其他组件通过 useModuleConfig() 消费。

import { quotaRoutes, useQuotaConfig } from '@quotapanel/quota/src/config'

import type { RouteRecordRaw } from 'vue-router'

// 版本类型（Pro / 社区版）：Pro 构建为 'pro'；发布到 Open 版时下面的 TYPE_PRO
// 条件块会被裁剪移除，回落为 'community'（社区版）。
const editionState: { value: 'pro' | 'community' } = { value: 'community' }


export interface NavItem {
  path: string
  iconName: string
  label: string
}

export interface ModuleOption {
  value: string
  label: string
  iconName: string
  defaultRoute: string
  isFullscreen: boolean
}

export interface ModuleConfig {
  navItems: NavItem[]
  moduleOptions: ModuleOption[]
  isFullscreen: (path: string) => boolean
  resolveModuleRoute: (value: string) => string
  editionKey: 'pro' | 'community'
}

/**
 * 返回纯静态路由，不依赖 i18n，可在模块加载时调用（router/index.ts）。
 */
export function getModuleRoutes(): RouteRecordRaw[] {
  const routes: RouteRecordRaw[] = []

  routes.push({ path: '/quota', redirect: '/quota/account' }, ...quotaRoutes)

  // 兜底：未匹配路径（含根路径）进入额度主页（仅额度模块编入时生效）
  routes.push({ path: '/:pathMatch(.*)*', redirect: '/quota/account' })

  return routes
}

/**
 * 返回带 i18n 标签的导航配置，必须在 Vue setup 上下文中调用（App.vue）。
 */
export function useModuleConfig(): ModuleConfig {
  const navItems: NavItem[] = []
  const moduleOptions: ModuleOption[] = []

  const quotaCfg = useQuotaConfig()
  navItems.push(...quotaCfg.navItems)
  moduleOptions.push(quotaCfg.moduleOption)

  function isFullscreen(path: string): boolean {
    return moduleOptions.some((m) => m.isFullscreen && path.startsWith('/' + m.value))
  }

  function resolveModuleRoute(value: string): string {
    const mod = moduleOptions.find((m) => m.value === value)
    return mod ? mod.defaultRoute : '/'
  }

  return { navItems, moduleOptions, isFullscreen, resolveModuleRoute, editionKey: editionState.value }
}

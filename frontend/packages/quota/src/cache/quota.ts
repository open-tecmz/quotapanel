import { ref } from 'vue'
import type { QuotaAccount, QuotaProviderInfo, QuotaSnapshot } from '../api/main'

/**
 * 额度页会话级缓存。
 *
 * 额度页是路由组件，切到其他页面会被卸载。把账号 / 额度存在模块级 ref 里，
 * 切回来时可以先用上次的数据渲染，再静默补齐最新结果，避免每次进入都先闪一下空状态。
 * 仅在内存中保留，应用重启后自然失效。
 */
export const quotaCache = {
  providers: ref<QuotaProviderInfo[]>([]),
  accounts: ref<QuotaAccount[]>([]),
  snapshots: ref<Record<number, QuotaSnapshot>>({}),
  // 首次进入时先按加载中处理，避免在账号数据回来前闪出「还没有订阅账号」。
  loadingAccounts: ref(true),
  queryingAll: ref(false),
  // 本会话是否已拿到全量额度结果（自动查询完成或收到后端轮询推送）。
  hasQueriedAll: ref(false),
  // 自动全量查询的实际触发次数，供测试断言「切回页面不重复查询」。
  queryAllCount: ref(0),
}

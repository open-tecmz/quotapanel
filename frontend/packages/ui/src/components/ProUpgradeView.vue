<template>
  <div class="pro-upgrade-view">
    <div class="pro-upgrade-inner">
      <!-- 图标 -->
      <div class="pro-icon-wrap">
        <Crown :size="36" class="pro-icon" aria-hidden="true" />
      </div>

      <!-- 标题 & 描述 -->
      <div class="pro-title">{{ title }} · Pro 专属功能</div>
      <div class="pro-desc">{{ desc }}</div>

      <!-- 特性列表 -->
      <div v-if="features.length" class="pro-features">
        <div v-for="feat in features" :key="feat" class="pro-feature-item">
          <Check :size="14" class="pro-feature-check" aria-hidden="true" />
          <span>{{ feat }}</span>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="pro-actions">
        <a-button type="primary" size="large" @click="doDownload">
          <div class="inline-flex items-center gap-2">
            <Download :size="16" aria-hidden="true" />
            下载 Pro 版本
          </div>
        </a-button>
        <a-button size="large" @click="doLearnMore">了解更多</a-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Check, Crown, Download } from 'lucide-vue-next'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { main } from '../api/call'

const props = withDefaults(
  defineProps<{
    title?: string
    desc?: string
    features?: string[]
  }>(),
  {
    title: '功能',
    desc: '此功能仅 Pro 版本支持，升级 Pro 解锁全部高级功能。',
    features: () => [],
  }
)

const websiteGithub = ref('')
const website = ref('')

onMounted(async () => {
  try {
    const cfg = await main.Call('setting.getAppConfig')
    websiteGithub.value = (cfg?.websiteGithub as string) || ''
    website.value = (cfg?.website as string) || ''
  } catch {}
})

function doDownload() {
  const url = websiteGithub.value || website.value
  if (url) BrowserOpenURL(url)
}

function doLearnMore() {
  const url = website.value || websiteGithub.value
  if (url) BrowserOpenURL(url)
}
</script>

<style scoped>
.pro-upgrade-view {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
  padding: 40px 24px;
}

.pro-upgrade-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.pro-icon-wrap {
  width: 72px;
  height: 72px;
  border-radius: 20px;
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 24px rgba(245, 158, 11, 0.3);
}

.pro-icon {
  color: #fff;
}

.pro-title {
  font-size: 20px;
  font-weight: 700;
  color: #1c1c1e;
  line-height: 1.3;
}

.pro-desc {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.6;
  max-width: 320px;
}

.pro-features {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 16px 20px;
  text-align: left;
}

.pro-feature-item {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: #374151;
}

.pro-feature-check {
  color: #018b8d;
  flex-shrink: 0;
}

.pro-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: center;
  margin-top: 4px;
}
</style>

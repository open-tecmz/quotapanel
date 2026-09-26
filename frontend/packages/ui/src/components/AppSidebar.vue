<script lang="ts" setup>
import { BedDouble, BookOpen, CalendarClock, Globe, MonitorPlay, Pin, Server, Settings, Wallet } from 'lucide-vue-next'
import type { Component } from 'vue'
import { useI18n } from 'vue-i18n'

interface NavItem {
  path: string
  iconName: string
  label: string
}

const props = defineProps<{
  navItems: NavItem[]
  isActive: (path: string) => boolean
  disabled?: boolean
  showPin?: boolean
  pinned?: boolean
}>()

const emit = defineEmits<{
  navigate: [path: string]
  togglePin: []
}>()

const { t } = useI18n()

const iconMap: Record<string, Component> = {
  Globe,
  Settings,
  BookOpen,
  BedDouble,
  Server,
  CalendarClock,
  MonitorPlay,
  Wallet,
}

function doNav(path: string) {
  if (props.disabled) return
  emit('navigate', path)
}
</script>

<template>
  <div class="app-sidebar">
    <div class="sidebar-tabs">
      <button
        v-for="item in navItems"
        :key="item.path"
        class="tab-button"
        :class="{ active: isActive(item.path), locked: disabled }"
        :title="disabled ? t('sidebar.disabledTitle') : item.label"
        @click="doNav(item.path)"
      >
        <component :is="iconMap[item.iconName]" :size="20" aria-hidden="true" />
        <span class="tab-label">{{ item.label }}</span>
      </button>

      <div class="flex-1"></div>

      <slot name="bottom" />

      <button
        v-if="showPin"
        class="tab-button pin-btn"
        :class="{ active: pinned }"
        :title="pinned ? t('sidebar.unpin') : t('sidebar.pin')"
        @click="emit('togglePin')"
      >
        <Pin :size="18" />
        <span class="tab-label">{{ t('sidebar.pinLabel') }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.app-sidebar {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  width: 100px;
  display: flex;
  flex-direction: column;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  border-right: 1px solid rgba(var(--token-neutral-200-rgb), 0.6);
  padding: 16px 12px;
  z-index: 10;
}

:global(.dark .app-sidebar) {
  background: rgba(var(--token-neutral-800-rgb), 0.8);
  border-right-color: rgba(var(--token-neutral-700-rgb), 0.6);
}

.sidebar-tabs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  height: 100%;
}

.tab-button,
:slotted(.tab-button) {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 8px;
  width: 100%;
  border: none;
  background: transparent;
  color: var(--token-neutral-500);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 13px;
  font-weight: 500;
}

.tab-button svg,
:slotted(.tab-button) svg {
  flex-shrink: 0;
  min-width: 20px;
}

.tab-button:hover:not(.locked):not(.active),
:slotted(.tab-button:hover) {
  background: rgba(var(--token-primary-rgb), 0.1);
  color: var(--token-primary-500);
}

.tab-button.active {
  background: linear-gradient(135deg, var(--token-primary-500) 0%, var(--token-primary-600) 100%);
  color: white;
  box-shadow: 0 4px 12px rgba(var(--token-primary-rgb), 0.3);
}

.tab-button.locked {
  cursor: not-allowed;
  opacity: 0.4;
}

:global(.dark .app-sidebar .tab-button) {
  color: var(--token-neutral-400);
}

:global(.dark .app-sidebar .tab-button:hover:not(.locked):not(.active)) {
  background: rgba(var(--token-primary-rgb), 0.15);
  color: var(--token-primary-600);
}

:global(.dark .app-sidebar .tab-button.active) {
  background: linear-gradient(135deg, var(--token-primary-500) 0%, var(--token-primary-400) 100%);
  color: white;
  box-shadow: 0 4px 12px rgba(var(--token-primary-rgb), 0.4);
}

.tab-label,
:slotted(.tab-label) {
  display: block;
  max-width: 100%;
  overflow: hidden;
  font-size: 13px;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Pin button — distinct amber toggle style */
.pin-btn.active {
  background: rgba(245, 158, 11, 0.12);
  color: var(--token-warning-strong);
  border: 1px solid rgba(245, 158, 11, 0.4);
  box-shadow: none;
}

:global(.dark .app-sidebar .pin-btn.active) {
  background: rgba(245, 158, 11, 0.15);
  color: var(--token-warning-light);
  border: 1px solid rgba(251, 191, 36, 0.4);
  box-shadow: none;
}
</style>

<template>
  <div class="flex items-center justify-between gap-4">
    <div class="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2 shrink-0">
      <slot name="icon" />
      {{ title }}
    </div>
    <div v-if="$slots.extra" class="flex items-center gap-2">
      <slot name="extra" />
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
}>()
</script>

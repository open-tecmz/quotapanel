<script lang="ts">
export interface ParamType {
  name: string
  type: 'text' | 'select'
  defaultValue: string
  options?: string[]
}
</script>

<script setup lang="ts">
import { computed } from 'vue'
import { Plus, Trash2 } from 'lucide-vue-next'

const props = defineProps<{ modelValue: ParamType[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: ParamType[]] }>()

const list = computed(() => props.modelValue ?? [])

function update(index: number, changes: Partial<ParamType>) {
  const next = list.value.map((item, i) => (i === index ? { ...item, ...changes } : item))
  emit('update:modelValue', next)
}

function addRow() {
  emit('update:modelValue', [...list.value, { name: '', type: 'text', defaultValue: '', options: [] }])
}

function removeRow(index: number) {
  emit(
    'update:modelValue',
    list.value.filter((_, i) => i !== index)
  )
}

function updateOptions(index: number, raw: string) {
  const options = raw
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  update(index, { options })
}
</script>

<template>
  <div class="space-y-2">
    <div v-if="list.length === 0" class="py-2 text-sm text-gray-400">暂无参数</div>
    <div
      v-for="(item, index) in list"
      :key="index"
      class="flex flex-col gap-2 rounded-xl border border-gray-200 bg-gray-50 p-3 transition-colors hover:border-blue-200 hover:bg-blue-50/30"
    >
      <div class="flex items-center gap-2">
        <a-input
          :value="item.name"
          placeholder="参数名（英文）"
          class="flex-1 font-mono"
          @change="
            update(index, {
              name: ($event.target as HTMLInputElement)?.value ?? '',
            })
          "
        />
        <a-select :value="item.type" class="w-24" @change="(v: any) => update(index, { type: v })">
          <a-select-option value="text">文本</a-select-option>
          <a-select-option value="select">下拉</a-select-option>
        </a-select>
        <a-button danger class="inline-flex items-center" @click="removeRow(index)">
          <Trash2 class="h-4 w-4" aria-hidden="true" />
        </a-button>
      </div>
      <div class="flex items-center gap-2">
        <span class="w-16 shrink-0 text-xs text-gray-500">默认值</span>
        <a-input
          :value="item.defaultValue"
          placeholder="留空则无默认值"
          class="flex-1"
          @change="
            update(index, {
              defaultValue: ($event.target as HTMLInputElement)?.value ?? '',
            })
          "
        />
      </div>
      <div v-if="item.type === 'select'" class="flex items-start gap-2">
        <span class="w-16 shrink-0 pt-1 text-xs text-gray-500">选项</span>
        <a-textarea
          :value="(item.options || []).join('\n')"
          :rows="3"
          placeholder="每行一个选项"
          class="flex-1 font-mono"
          @change="updateOptions(index, ($event.target as HTMLTextAreaElement)?.value ?? '')"
        />
      </div>
    </div>
    <a-button @click="addRow">
      <div class="inline-flex items-center gap-1">
        <Plus class="h-4 w-4" aria-hidden="true" />
        添加参数
      </div>
    </a-button>
  </div>
</template>

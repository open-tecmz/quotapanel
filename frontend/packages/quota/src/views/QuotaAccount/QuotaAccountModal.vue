<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { QuotaAccount, QuotaProviderInfo } from '../../api/main'
import { main } from '../../api/main'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  providers: QuotaProviderInfo[]
  account?: QuotaAccount | null
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  saved: [account: QuotaAccount]
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)

const formState = reactive({
  provider: '',
  name: '',
  key: '',
  note: '',
})

const isEdit = computed(() => !!props.account)

const rules = computed(() => ({
  provider: [{ required: true, message: t('quota.account.chooseProvider'), trigger: 'change' }],
  key:
    isEdit.value || currentProvider.value?.mode === 'browser'
      ? []
      : [{ required: true, message: t('quota.account.enterKey'), trigger: 'blur' }],
}))

watch(
  () => props.open,
  (val) => {
    if (!val) return
    if (props.account) {
      formState.provider = props.account.provider
      formState.name = props.account.name
      formState.key = ''
      formState.note = props.account.note
    } else {
      formState.provider = props.providers[0]?.id ?? ''
      formState.name = ''
      formState.key = ''
      formState.note = ''
    }
    formRef.value?.clearValidate()
  }
)

const currentProvider = computed(() => props.providers.find((p) => p.id === formState.provider))

// Match a provider against the search keyword (name / id / description).
function matchesProvider(provider: QuotaProviderInfo, keyword: string): boolean {
  const query = keyword.trim().toLowerCase()
  if (!query) return true
  return [provider.name, provider.id, provider.description].some((field) =>
    String(field || '')
      .toLowerCase()
      .includes(query)
  )
}

// Custom filter for the searchable provider select.
function filterProvider(input: string, option: unknown): boolean {
  const value = (option as { value?: string } | undefined)?.value
  const provider = props.providers.find((item) => item.id === value)
  if (!provider) return true
  return matchesProvider(provider, input)
}

function handleCancel() {
  emit('update:open', false)
}

async function handleSubmit() {
  try {
    await formRef.value?.validateFields()
  } catch {
    return
  }
  submitting.value = true
  try {
    let saved: QuotaAccount
    if (props.account) {
      saved = await main.Call('quota.updateAccount', {
        id: props.account.id,
        name: formState.name,
        key: formState.key,
        note: formState.note,
      })
      message.success(t('quota.account.saved'))
    } else {
      saved = await main.Call('quota.addAccount', {
        provider: formState.provider,
        name: formState.name,
        key: formState.key,
        note: formState.note,
      })
      message.success(t('quota.account.added'))
    }
    emit('update:open', false)
    emit('saved', saved)
  } catch (e: unknown) {
    message.error(e instanceof Error ? e.message : String(e))
  } finally {
    submitting.value = false
  }
}

function testFill(data: { provider?: string; name?: string; key?: string; note?: string }) {
  if (data.provider !== undefined) formState.provider = data.provider
  if (data.name !== undefined) formState.name = data.name
  if (data.key !== undefined) formState.key = data.key
  if (data.note !== undefined) formState.note = data.note
}

async function testSubmit() {
  await handleSubmit()
}

function testSearch(keyword: string): string[] {
  return props.providers.filter((provider) => matchesProvider(provider, keyword)).map((provider) => provider.id)
}

defineExpose({ testFill, testSubmit, testSearch })
</script>

<template>
  <a-modal
    :open="open"
    :title="isEdit ? t('quota.account.editTitle') : t('quota.account.addTitle')"
    width="min(600px, 90vw)"
    :confirm-loading="submitting"
    :ok-text="isEdit ? t('quota.account.save') : t('quota.account.add')"
    :cancel-text="t('quota.account.cancel')"
    @ok="handleSubmit"
    @cancel="handleCancel"
  >
    <a-form ref="formRef" :model="formState" :rules="rules" layout="vertical" class="mt-4">
      <a-form-item :label="t('quota.account.provider')" name="provider">
        <a-select
          v-model:value="formState.provider"
          show-search
          :disabled="isEdit"
          :filter-option="filterProvider"
          :placeholder="t('quota.account.chooseProvider')"
        >
          <a-select-option v-for="p in providers" :key="p.id" :value="p.id">
            {{ p.name }}（{{ p.mode === 'api' ? t('quota.account.apiMode') : t('quota.account.browserMode') }}）
          </a-select-option>
        </a-select>
        <div v-if="currentProvider" class="field-hint">
          {{ currentProvider.description }}
        </div>
      </a-form-item>

      <a-form-item :label="t('quota.account.name')" name="name">
        <a-input
          v-model:value="formState.name"
          :maxlength="50"
          :placeholder="currentProvider?.name || t('quota.account.customName')"
        />
      </a-form-item>

      <a-form-item v-if="currentProvider?.mode !== 'browser'" :label="currentProvider?.keyLabel || 'Key'" name="key">
        <a-input-password
          v-model:value="formState.key"
          :placeholder="isEdit ? t('quota.account.keepKey') : currentProvider?.keyHint || t('quota.account.enterKey')"
        />
      </a-form-item>

      <a-form-item :label="t('quota.account.note')" name="note">
        <a-textarea
          v-model:value="formState.note"
          :rows="2"
          :maxlength="200"
          :placeholder="t('quota.account.noteHint')"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<style scoped>
.field-hint {
  font-size: 11px;
  color: #9ca3af;
  margin-top: 4px;
  line-height: 1.5;
}
:global(.dark .field-hint) {
  color: #9ca3af;
}
</style>

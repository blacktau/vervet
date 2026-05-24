<template>
  <n-form label-placement="top" :show-feedback="false">
    <n-form-item :label="$t('serverPane.dialogs.server.auth.scram.username')">
      <n-input
        :value="fields.username"
        data-testid="scram-username"
        @update:value="(v: string) => update({ username: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.scram.password')">
      <n-input
        :value="fields.password"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ password: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.scram.authSource')">
      <n-input
        :value="fields.authSource ?? ''"
        :placeholder="$t('serverPane.dialogs.server.auth.scram.authSourcePlaceholder')"
        @update:value="(v: string) => update({ authSource: v || undefined })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.scram.mechanism')">
      <n-select
        :value="fields.mechanism ?? 'auto'"
        :options="mechanismOptions"
        @update:value="(v: ScramAuth['mechanism']) => update({ mechanism: v })"
      />
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { parseScram, serialiseScram, type ScramAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const { t } = useI18n()

const fields = computed<ScramAuth>(() => parseScram(props.uri))

const mechanismOptions = computed(() => [
  { label: t('serverPane.dialogs.server.auth.scram.mechanismAuto'), value: 'auto' },
  { label: 'SCRAM-SHA-256', value: 'SCRAM-SHA-256' },
  { label: 'SCRAM-SHA-1', value: 'SCRAM-SHA-1' },
])

function update(patch: Partial<ScramAuth>): void {
  emit('update:uri', serialiseScram(props.uri, { ...fields.value, ...patch }))
}

const warnings = computed<string[]>(() => {
  const out: string[] = []
  if (!fields.value.username) {
    out.push(t('serverPane.dialogs.server.auth.warnings.missingUsername'))
  }
  return out
})

defineExpose({ warnings })
</script>

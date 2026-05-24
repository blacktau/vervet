<template>
  <n-form label-placement="top" :show-feedback="false">
    <n-form-item :label="$t('serverPane.dialogs.server.auth.plain.username')">
      <n-input
        :value="fields.username"
        data-testid="plain-username"
        @update:value="(v: string) => update({ username: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.plain.password')">
      <n-input
        :value="fields.password"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ password: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.plain.authSource')">
      <n-input
        :value="fields.authSource ?? '$external'"
        :placeholder="$t('serverPane.dialogs.server.auth.plain.authSourcePlaceholder')"
        @update:value="(v: string) => update({ authSource: v || undefined })"
      />
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { parsePlain, serialisePlain, type PlainAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const { t } = useI18n()

const fields = computed<PlainAuth>(() => parsePlain(props.uri))

function update(patch: Partial<PlainAuth>): void {
  emit('update:uri', serialisePlain(props.uri, { ...fields.value, ...patch }))
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

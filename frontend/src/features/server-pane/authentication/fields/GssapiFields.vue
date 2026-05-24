<template>
  <n-form label-placement="top" :show-feedback="false">
    <n-form-item :label="$t('serverPane.dialogs.server.auth.gssapi.principal')">
      <n-input
        :value="fields.principal"
        :placeholder="$t('serverPane.dialogs.server.auth.gssapi.principalPlaceholder')"
        data-testid="gssapi-principal"
        @update:value="(v: string) => update({ principal: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.gssapi.serviceName')">
      <n-input
        :value="fields.serviceName ?? ''"
        :placeholder="$t('serverPane.dialogs.server.auth.gssapi.serviceNamePlaceholder')"
        @update:value="(v: string) => update({ serviceName: v || undefined })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.gssapi.canonicalize')">
      <n-select
        :value="fields.canonicalize ?? 'none'"
        :options="canonicalizeOptions"
        @update:value="(v: string) => update({ canonicalize: v === 'none' ? undefined : (v as GssapiAuth['canonicalize']) })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.gssapi.serviceRealm')">
      <n-input
        :value="fields.serviceRealm ?? ''"
        @update:value="(v: string) => update({ serviceRealm: v || undefined })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.gssapi.password')">
      <n-input
        :value="fields.password ?? ''"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ password: v || undefined })"
      />
      <template #feedback>
        {{ $t('serverPane.dialogs.server.auth.gssapi.passwordHelp') }}
      </template>
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { parseGssapi, serialiseGssapi, type GssapiAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const { t } = useI18n()

const fields = computed<GssapiAuth>(() => parseGssapi(props.uri))

const canonicalizeOptions = computed(() => [
  { label: t('serverPane.dialogs.server.auth.gssapi.canonicalizeNone'), value: 'none' },
  { label: t('serverPane.dialogs.server.auth.gssapi.canonicalizeForward'), value: 'forward' },
  {
    label: t('serverPane.dialogs.server.auth.gssapi.canonicalizeForwardReverse'),
    value: 'forwardAndReverse',
  },
])

function update(patch: Partial<GssapiAuth>): void {
  emit('update:uri', serialiseGssapi(props.uri, { ...fields.value, ...patch }))
}

const warnings = computed<string[]>(() => {
  const out: string[] = []
  if (!fields.value.principal) {
    out.push(t('serverPane.dialogs.server.auth.warnings.missingPrincipal'))
  }
  return out
})

defineExpose({ warnings })
</script>

<template>
  <n-form label-placement="top">
    <n-form-item
      :label="$t('serverPane.dialogs.server.auth.gssapi.principal')"
      :validation-status="principalTouched && !fields.principal ? 'warning' : undefined"
      :feedback="principalTouched && !fields.principal ? $t('serverPane.dialogs.server.auth.warnings.missingPrincipal') : ''">
      <n-input
        :value="fields.principal"
        :placeholder="$t('serverPane.dialogs.server.auth.gssapi.principalPlaceholder')"
        data-testid="gssapi-principal"
        @update:value="(v: string) => update({ principal: v })"
        @blur="principalTouched = true"
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
    <n-form-item>
      <template #label>
        <span class="label-with-help">
          {{ $t('serverPane.dialogs.server.auth.gssapi.password') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon :component="InformationCircleIcon" size="14" />
            </template>
            {{ $t('serverPane.dialogs.server.auth.gssapi.passwordHelp') }}
          </n-tooltip>
        </span>
      </template>
      <n-input
        :value="fields.password ?? ''"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ password: v || undefined })"
      />
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { parseGssapi, serialiseGssapi, type GssapiAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const i18n = useI18n()

const fields = computed<GssapiAuth>(() => parseGssapi(props.uri))
const principalTouched = ref(false)

const canonicalizeOptions = computed(() => [
  { label: i18n.t('serverPane.dialogs.server.auth.gssapi.canonicalizeNone'), value: 'none' },
  { label: i18n.t('serverPane.dialogs.server.auth.gssapi.canonicalizeForward'), value: 'forward' },
  {
    label: i18n.t('serverPane.dialogs.server.auth.gssapi.canonicalizeForwardReverse'),
    value: 'forwardAndReverse',
  },
])

function update(patch: Partial<GssapiAuth>): void {
  emit('update:uri', serialiseGssapi(props.uri, { ...fields.value, ...patch }))
}
</script>

<style scoped>
.label-with-help {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}
</style>

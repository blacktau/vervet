<template>
  <div class="authentication-panel">
    <n-form-item :label="$t('serverPane.dialogs.server.authMethod')" :show-feedback="false">
      <n-select :value="method" :options="pickerOptions" @update:value="onMethodChange" />
    </n-form-item>

    <component
      :is="fieldsComponent"
      :uri="uri"
      :oidc-config="oidcConfig"
      @update:uri="(v: string) => $emit('update:uri', v)"
      @update:oidc-config="(v: OIDCConfig) => $emit('update:oidcConfig', v)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { AuthMethod, OIDCConfig } from '@/types/ConnectionConfig'
import {
  detectAuthFromUri,
  setAuthMechanism,
  type SyncableAuthMechanism,
} from '../connectionStrings'
import NoneFields from './fields/NoneFields.vue'
import ScramFields from './fields/ScramFields.vue'
import X509Fields from './fields/X509Fields.vue'
import OidcFields from './fields/OidcFields.vue'
import AwsFields from './fields/AwsFields.vue'
import GssapiFields from './fields/GssapiFields.vue'
import PlainFields from './fields/PlainFields.vue'

type PickerValue = AuthMethod | 'auto'

const props = defineProps<{
  uri: string
  method: PickerValue
  oidcConfig: OIDCConfig
}>()

const emit = defineEmits<{
  (e: 'update:uri', value: string): void
  (e: 'update:method', value: PickerValue): void
  (e: 'update:oidcConfig', value: OIDCConfig): void
}>()

const { t } = useI18n()

const pickerOptions = computed(() => [
  { label: t('serverPane.dialogs.server.auth.picker.auto'), value: 'auto' },
  { label: t('serverPane.dialogs.server.auth.picker.none'), value: 'none' },
  { label: t('serverPane.dialogs.server.auth.picker.password'), value: 'password' },
  { label: t('serverPane.dialogs.server.auth.picker.x509'), value: 'x509' },
  { label: t('serverPane.dialogs.server.auth.picker.oidc'), value: 'oidc' },
  { label: t('serverPane.dialogs.server.auth.picker.aws'), value: 'aws' },
  { label: t('serverPane.dialogs.server.auth.picker.gssapi'), value: 'gssapi' },
  { label: t('serverPane.dialogs.server.auth.picker.plain'), value: 'plain' },
])

const effective = computed<AuthMethod>(() => {
  if (props.method === 'auto') {
    return detectAuthFromUri(props.uri).authMethod
  }
  return props.method
})

const fieldsComponent = computed(() => {
  switch (effective.value) {
    case 'none':
      return NoneFields
    case 'password':
      return ScramFields
    case 'x509':
      return X509Fields
    case 'oidc':
      return OidcFields
    case 'aws':
      return AwsFields
    case 'gssapi':
      return GssapiFields
    case 'plain':
      return PlainFields
    default:
      return NoneFields
  }
})

const SYNCABLE: Partial<Record<AuthMethod, SyncableAuthMechanism>> = {
  x509: 'MONGODB-X509',
  oidc: 'MONGODB-OIDC',
  aws: 'MONGODB-AWS',
  gssapi: 'GSSAPI',
  plain: 'PLAIN',
}

function onMethodChange(next: PickerValue): void {
  emit('update:method', next)
  if (next === 'auto') {
    return
  }
  const mechanism = next === 'none' || next === 'password' ? null : (SYNCABLE[next] ?? null)
  const newUri = setAuthMechanism(props.uri, mechanism)
  if (newUri !== props.uri) {
    emit('update:uri', newUri)
  }
}

</script>

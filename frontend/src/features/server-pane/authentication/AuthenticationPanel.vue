<template>
  <div class="authentication-panel">
    <n-form-item :label="$t('serverPane.dialogs.server.authMethod')">
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
import { clearAuthState, setAuthMechanism, type SyncableAuthMechanism } from '../connectionStrings'
import NoneFields from './fields/NoneFields.vue'
import ScramFields from './fields/ScramFields.vue'
import X509Fields from './fields/X509Fields.vue'
import OidcFields from './fields/OidcFields.vue'
import AwsFields from './fields/AwsFields.vue'
import GssapiFields from './fields/GssapiFields.vue'
import PlainFields from './fields/PlainFields.vue'

const props = defineProps<{
  uri: string
  method: AuthMethod
  oidcConfig: OIDCConfig
}>()

const emit = defineEmits<{
  (e: 'update:uri', value: string): void
  (e: 'update:method', value: AuthMethod): void
  (e: 'update:oidcConfig', value: OIDCConfig): void
}>()

const { t } = useI18n()

const pickerOptions = computed(() => [
  { label: t('serverPane.dialogs.server.auth.picker.none'), value: 'none' },
  { label: t('serverPane.dialogs.server.auth.picker.password'), value: 'password' },
  { label: t('serverPane.dialogs.server.auth.picker.x509'), value: 'x509' },
  { label: t('serverPane.dialogs.server.auth.picker.oidc'), value: 'oidc' },
  { label: t('serverPane.dialogs.server.auth.picker.aws'), value: 'aws' },
  { label: t('serverPane.dialogs.server.auth.picker.gssapi'), value: 'gssapi' },
  { label: t('serverPane.dialogs.server.auth.picker.plain'), value: 'plain' },
])

const fieldsComponent = computed(() => {
  switch (props.method) {
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

const USERINFO_METHODS: ReadonlySet<AuthMethod> = new Set(['password', 'plain', 'gssapi'])

function onMethodChange(next: AuthMethod): void {
  emit('update:method', next)
  const mechanism = next === 'none' || next === 'password' ? null : (SYNCABLE[next] ?? null)
  const cleared = clearAuthState(props.uri, { stripUserinfo: !USERINFO_METHODS.has(next) })
  const newUri = mechanism === null ? cleared : setAuthMechanism(cleared, mechanism)
  if (newUri !== props.uri) {
    emit('update:uri', newUri)
  }
}

</script>

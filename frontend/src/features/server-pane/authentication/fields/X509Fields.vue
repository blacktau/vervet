<template>
  <n-form label-placement="top" :show-feedback="false">
    <n-form-item :label="$t('serverPane.dialogs.server.auth.x509.certFile')">
      <n-input-group>
        <n-input
          :value="fields.certFile"
          :placeholder="$t('serverPane.dialogs.server.auth.x509.certFilePlaceholder')"
          data-testid="x509-cert-file"
          @update:value="(v: string) => update({ certFile: v })"
        />
        <n-button @click="browse">
          {{ $t('serverPane.dialogs.server.auth.x509.certFileBrowse') }}
        </n-button>
      </n-input-group>
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.x509.certPassphrase')">
      <n-input
        :value="fields.certPassphrase ?? ''"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ certPassphrase: v || undefined })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.x509.usernameOverride')">
      <n-input
        :value="fields.usernameOverride ?? ''"
        @update:value="(v: string) => update({ usernameOverride: v || undefined })"
      />
      <template #feedback>
        {{ $t('serverPane.dialogs.server.auth.x509.usernameOverrideHelp') }}
      </template>
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.x509.authSource')">
      <n-input
        :value="fields.authSource ?? '$external'"
        :placeholder="$t('serverPane.dialogs.server.auth.x509.authSourcePlaceholder')"
        @update:value="(v: string) => update({ authSource: v || undefined })"
      />
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { parseX509, serialiseX509, type X509Auth } from '../authUri'
import { SelectFile } from 'wailsjs/go/api/FilesProxy'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const { t } = useI18n()

const fields = computed<X509Auth>(() => parseX509(props.uri))

function update(patch: Partial<X509Auth>): void {
  emit('update:uri', serialiseX509(props.uri, { ...fields.value, ...patch }))
}

async function browse(): Promise<void> {
  const result = await SelectFile(t('serverPane.dialogs.server.auth.x509.certFile'), [
    { displayName: 'PEM files', pattern: '*.pem;*.crt;*.key' },
    { displayName: 'All files', pattern: '*' },
  ])
  if (result.isSuccess && result.data) {
    update({ certFile: result.data })
  }
}

const warnings = computed<string[]>(() => {
  const out: string[] = []
  if (!fields.value.certFile) {
    out.push(t('serverPane.dialogs.server.auth.warnings.missingCertFile'))
  }
  return out
})

defineExpose({ warnings })
</script>

<template>
  <n-form :show-require-mark="false" label-placement="top">
    <n-form-item :label="$t('serverPane.dialogs.server.oidcSignInBehaviour')">
      <n-select
        :value="signInBehaviour"
        :options="signInOptions"
        @update:value="(v: SignInBehaviour) => onBehaviour(v)"
      />
    </n-form-item>
    <n-collapse>
      <n-collapse-item :title="$t('serverPane.dialogs.server.oidcAdvanced')" name="adv">
        <n-form-item :label="$t('serverPane.dialogs.server.oidcProviderUrl')">
          <n-input
            :value="local.providerUrl"
            placeholder="Auto-detected from server"
            @update:value="(v: string) => patch({ providerUrl: v })"
          />
        </n-form-item>
        <n-form-item :label="$t('serverPane.dialogs.server.oidcClientId')">
          <n-input
            :value="local.clientId"
            placeholder="Auto-detected from server"
            @update:value="(v: string) => patch({ clientId: v })"
          />
        </n-form-item>
        <n-form-item :label="$t('serverPane.dialogs.server.oidcScopes')">
          <n-input
            :value="local.scopes?.join(', ') ?? ''"
            placeholder="Auto-detected from server"
            @update:value="onScopes"
          />
        </n-form-item>
        <n-form-item>
          <n-checkbox
            :checked="local.workloadIdentity"
            @update:checked="(v: boolean) => patch({ workloadIdentity: v })">
            {{ $t('serverPane.dialogs.server.oidcWorkloadIdentityDesc') }}
          </n-checkbox>
        </n-form-item>
      </n-collapse-item>
    </n-collapse>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { OIDCConfig } from '@/types/ConnectionConfig'
import {
  applySignInBehaviour,
  signInBehaviourFromConfig,
  type SignInBehaviour,
} from '../../connectionStrings'

const props = defineProps<{
  uri: string
  oidcConfig: OIDCConfig
}>()

const emit = defineEmits<{
  (e: 'update:oidcConfig', value: OIDCConfig): void
}>()

const { t } = useI18n()

const local = computed<OIDCConfig>(() => props.oidcConfig)

const signInBehaviour = computed<SignInBehaviour>(() => signInBehaviourFromConfig(local.value))

const signInOptions = computed(() => [
  { label: t('serverPane.dialogs.server.oidcSignInOpenBrowser'), value: 'openBrowser' },
  { label: t('serverPane.dialogs.server.oidcSignInForceAccountPicker'), value: 'forceAccountPicker' },
  { label: t('serverPane.dialogs.server.oidcSignInShowUrl'), value: 'showUrl' },
])

function patch(p: Partial<OIDCConfig>): void {
  emit('update:oidcConfig', { ...local.value, ...p })
}

function onBehaviour(b: SignInBehaviour): void {
  emit('update:oidcConfig', applySignInBehaviour(local.value, b))
}

function onScopes(v: string): void {
  patch({
    scopes: v
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
  })
}

</script>

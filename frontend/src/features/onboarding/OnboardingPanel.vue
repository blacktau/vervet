<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  detectAuthFromUri,
  getUriHost,
  parseUri,
} from '@/features/server-pane/connectionStrings.ts'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { useServerConnection } from '@/features/server-pane/useServerConnection.ts'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import type { RegisteredServerNode } from '@/features/server-pane/serverStore.ts'

const i18n = useI18n()

const serverStore = useServerStore()
const { connectToServer } = useServerConnection()

const uri = ref('')
const name = ref('')
const connecting = ref(false)
const lastError = ref<{ code: string; detail: string } | null>(null)

const nameTouched = ref(false)

watch(uri, () => {
  lastError.value = null
})

watch(uri, (next) => {
  if (nameTouched.value) {
    return
  }
  const host = getUriHost(next)
  if (host) {
    name.value = host
  }
})

const onNameInput = (value: string) => {
  nameTouched.value = true
  name.value = value
}

const uriValid = computed(() => {
  if (!uri.value) {
    return false
  }
  return parseUri(uri.value).success
})

const canConnect = computed(() => uriValid.value && !connecting.value)

const onConnect = async () => {
  if (!uriValid.value) {
    return
  }
  lastError.value = null
  connecting.value = true
  try {
    const detected = detectAuthFromUri(uri.value)
    const finalUri = detected.uri

    if (detected.authMethod !== 'oidc') {
      const testResult = await connectionsProxy.TestConnection(finalUri)
      if (!testResult.isSuccess) {
        lastError.value = {
          code: testResult.errorCode,
          detail: testResult.errorDetail,
        }
        return
      }
    }

    const existingIds = collectServerIds(serverStore.serverTree)

    const saveResult = await serverStore.saveServerWithConfig(
      name.value,
      '',
      '',
      { uri: finalUri, authMethod: detected.authMethod, oidcConfig: undefined },
    )
    if (!saveResult.success) {
      lastError.value = { code: 'saveFailed', detail: saveResult.msg ?? '' }
      return
    }

    const newId = findNewServerId(serverStore.serverTree, existingIds)
    if (!newId) {
      lastError.value = { code: 'saveFailed', detail: 'Saved server not found in tree' }
      return
    }
    await connectToServer(newId)
  } finally {
    connecting.value = false
  }
}

function collectServerIds(nodes: RegisteredServerNode[]): Set<string> {
  const ids = new Set<string>()
  const walk = (list: RegisteredServerNode[]) => {
    for (const node of list) {
      if (!node.isGroup) {
        ids.add(node.id)
      }
      if (node.children) {
        walk(node.children)
      }
    }
  }
  walk(nodes)
  return ids
}

function findNewServerId(
  nodes: RegisteredServerNode[],
  before: Set<string>,
): string | undefined {
  let found: string | undefined
  const walk = (list: RegisteredServerNode[]) => {
    for (const node of list) {
      if (!node.isGroup && !before.has(node.id)) {
        found = node.id
        return
      }
      if (node.children) {
        walk(node.children)
        if (found) {
          return
        }
      }
    }
  }
  walk(nodes)
  return found
}
</script>

<template>
  <div class="onboarding-wrapper flex-box-v">
    <div class="onboarding-card">
      <h1 class="title">{{ i18n.t('onboarding.welcomeTitle') }}</h1>
      <p class="subtitle">{{ i18n.t('onboarding.welcomeSubtitle') }}</p>

      <n-form label-placement="top">
        <n-form-item :label="i18n.t('onboarding.uriLabel')">
          <n-input
            v-model:value="uri"
            :placeholder="i18n.t('onboarding.uriPlaceholder')"
            data-test="uri-input"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
        </n-form-item>

        <n-form-item :label="i18n.t('onboarding.nameLabel')">
          <n-input
            :value="name"
            :placeholder="i18n.t('onboarding.namePlaceholder')"
            data-test="name-input"
            @update:value="onNameInput"
          />
        </n-form-item>

        <n-alert
          v-if="lastError"
          type="error"
          :closable="false"
          :title="i18n.t('onboarding.errorTitle')"
          data-test="error-alert"
        >
          <div>{{ i18n.t('errors.' + lastError.code) }}</div>
          <div v-if="lastError.detail" class="error-detail">{{ lastError.detail }}</div>
        </n-alert>

        <n-button
          :disabled="!canConnect"
          :loading="connecting"
          block
          size="large"
          type="primary"
          data-test="connect-btn"
          @click="onConnect"
        >
          {{ i18n.t('onboarding.connect') }}
        </n-button>

        <div class="advanced-link">
          <n-button text data-test="advanced-link">
            {{ i18n.t('onboarding.advanced') }}
          </n-button>
        </div>
      </n-form>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.onboarding-wrapper {
  height: 100%;
  justify-content: center;
  align-items: center;
  padding: 20px;
}

.onboarding-card {
  width: 100%;
  max-width: 480px;
}

.title {
  font-size: 24px;
  font-weight: 500;
  margin: 0 0 4px;
}

.subtitle {
  margin: 0 0 24px;
  opacity: 0.75;
}

.advanced-link {
  text-align: center;
  margin-top: 12px;
}

.error-detail {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.8;
}
</style>

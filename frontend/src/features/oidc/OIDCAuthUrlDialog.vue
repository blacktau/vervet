<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import * as runtime from 'wailsjs/runtime'
import * as oidcProxy from 'wailsjs/go/api/OIDCProxy'
import { useServerStore } from '@/features/server-pane/serverStore.ts'

const i18n = useI18n()
const message = useMessage()
const serverStore = useServerStore()

const show = ref(false)
const serverID = ref('')
const url = ref('')
const serverName = ref('')

const onAuthUrl = (payload: { serverID: string; url: string }) => {
  serverID.value = payload.serverID
  url.value = payload.url
  const server = serverStore.findServerById(payload.serverID)
  serverName.value = server?.name ?? payload.serverID
  show.value = true
}

const onConnected = (id: string) => {
  if (show.value && id === serverID.value) {
    show.value = false
  }
}

let unsubAuthUrl: (() => void) | undefined
let unsubConnected: (() => void) | undefined

onMounted(() => {
  unsubAuthUrl = runtime.EventsOn('oidc-auth-url', onAuthUrl)
  unsubConnected = runtime.EventsOn('connection-connected', onConnected)
})

onBeforeUnmount(() => {
  unsubAuthUrl?.()
  unsubConnected?.()
})

const onCopy = async () => {
  try {
    await navigator.clipboard.writeText(url.value)
    message.success(i18n.t('serverPane.dialogs.server.oidcAuthUrlCopied'))
  } catch {
    message.error('Clipboard unavailable. Select and copy manually.')
  }
}

const onCancel = async () => {
  await oidcProxy.CancelLogin(serverID.value)
  show.value = false
}
</script>

<template>
  <n-modal
    v-model:show="show"
    :closable="false"
    :mask-closable="false"
    :show-icon="false"
    :title="i18n.t('serverPane.dialogs.server.oidcAuthUrlTitle', { name: serverName })"
    preset="dialog"
    style="width: 600px"
    transform-origin="center"
  >
    <p>{{ i18n.t('serverPane.dialogs.server.oidcAuthUrlDescription') }}</p>
    <n-input :value="url" type="textarea" :autosize="{ minRows: 3, maxRows: 6 }" readonly />
    <template #action>
      <div class="flex-item-expand">
        <n-button :focusable="false" @click="onCopy">
          {{ i18n.t('serverPane.dialogs.server.oidcAuthUrlCopy') }}
        </n-button>
      </div>
      <n-button :focusable="false" @click="onCancel">
        {{ i18n.t('common.cancel') }}
      </n-button>
    </template>
  </n-modal>
</template>

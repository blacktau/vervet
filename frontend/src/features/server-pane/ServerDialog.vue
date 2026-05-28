<script lang="ts" setup>
import { type FormInst, type FormRules, useThemeVars } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { computed, nextTick, ref, watch } from 'vue'
import { every, includes, isEmpty } from 'lodash'
import { XCircleIcon } from '@heroicons/vue/24/outline'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { type RegisteredServerNode, useServerStore } from '@/features/server-pane/serverStore.ts'
import { useMessager, useNotifier } from '@/utils/dialog.ts'
import {
  parseUri,
  detectAuthFromUri,
  getUriHost,
} from '@/features/server-pane/connectionStrings.ts'
import { filterGroupMap } from '@/features/server-pane/helpers.ts'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import type { AuthMethod, OIDCConfig } from '@/types/ConnectionConfig'
import AuthenticationPanel from './authentication/AuthenticationPanel.vue'

type EditableRegisteredServer = {
  id: string
  name: string
  connectionString: string
  parentId: string
  colour: string
}

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const serverStore = useServerStore()
const browserStore = useDataBrowserStore()
const i18n = useI18n()

const editServerID = ref(dialogStore.serverDialogData?.serverId)
const tab = ref('general')
const serverColors = ref<string[]>([
  '',
  '#F75B52',
  '#F7A234',
  '#F7CE33',
  '#4ECF60',
  '#348CF7',
  '#B270D3',
])

const generalForm = ref<EditableRegisteredServer>({
  id: '',
  name: '',
  parentId: '',
  connectionString: '',
  colour: '',
})

const generalFormRef = ref<FormInst | undefined>(undefined)
const testing = ref(false)

const authPicker = ref<AuthMethod>('password')
const lastChangeSource = ref<'uri' | 'picker' | null>(null)
const nameWasEdited = ref(false)

const hintMechanismLabel = computed(() =>
  i18n.t(`serverPane.dialogs.server.auth.picker.${authPicker.value}`),
)

const oidcConfig = ref<OIDCConfig>({
  providerUrl: '',
  clientId: '',
  scopes: [],
  workloadIdentity: false,
  prompt: '',
  manualUrlMode: false,
})

const generalFormRules = () => {
  const requiredMsg = i18n.t('common.dialog.fieldRequired')
  const illegalChars = ['/', '\\']
  return {
    name: [
      { required: true, message: requiredMsg, trigger: 'input' },
      {
        validator: (rule, value) => {
          return every(illegalChars, (c) => !includes(value, c))
        },
        message: i18n.t('common.dialog.illegalCharacters'),
        trigger: 'input',
      },
    ],
    connectionString: [
      { required: true, message: requiredMsg, trigger: 'input' },
      {
        validator: (rule, value) => {
          const result = parseUri(value)
          if (!result.success) {
            return new Error(result.error!)
          }
          return true
        },
      },
    ],
  } as FormRules
}

const isEditMode = computed(() => dialogStore.serverDialogData?.mode === 'edit')

const closingConnection = computed(() => {
  if (!isEditMode.value) {
    return false
  }
  if (isEmpty(editServerID.value)) {
    return false
  }
  return browserStore.isConnected(editServerID.value)
})

const onSaveServer = async () => {
  await generalFormRef.value?.validate((err) => {
    if (err) {
      nextTick(() => (tab.value = 'general'))
    }
  })

  const cfg = {
    uri: generalForm.value.connectionString,
    authMethod: authPicker.value,
    oidcConfig: authPicker.value === 'oidc' ? { ...oidcConfig.value } : undefined,
  }

  const messager = useMessager()
  if (!isEditMode.value) {
    const result = await serverStore.saveServerWithConfig(
      generalForm.value.name,
      generalForm.value.parentId,
      generalForm.value.colour,
      cfg,
    )
    if (!result.success) {
      messager.error(result.msg || 'unknown error')
      return
    }
  } else {
    const result = await serverStore.updateServerWithConfig(
      editServerID.value || null,
      generalForm.value.name,
      generalForm.value.parentId,
      generalForm.value.colour,
      cfg,
    )
    if (!result.success) {
      messager.error(result.msg || 'unknown error')
      return
    }
  }

  messager.success(i18n.t('common.dialog.handleSuccess'))
  onClose()
}

const NEW_GROUP_SENTINEL = '__new_group__'

function syntheticGroupNode(id: string, name: string): RegisteredServerNode {
  return {
    id,
    name,
    isGroup: true,
    isCluster: false,
    isSrv: false,
    children: [],
    colour: '',
  }
}

const groupOptions = computed(() => {
  const realGroups: RegisteredServerNode[] = []
  for (const node of serverStore.serverTree) {
    const option = filterGroupMap(node)
    if (option) {
      realGroups.push(option)
    }
  }
  return [
    syntheticGroupNode(NEW_GROUP_SENTINEL, i18n.t('serverPane.dialogs.common.newGroup')),
    syntheticGroupNode('', i18n.t('serverPane.dialogs.common.noGroup')),
    ...realGroups,
  ]
})

const previousParentId = ref('')

function onParentChange(next: string) {
  if (next === NEW_GROUP_SENTINEL) {
    const prev = previousParentId.value
    generalForm.value.parentId = prev
    dialogStore.showNewDialog(DialogType.Group, {
      onCreated: (id: string) => {
        generalForm.value.parentId = id
        previousParentId.value = id
      },
    })
    return
  }
  previousParentId.value = next
}

const onClose = () => {
  dialogStore.hide(DialogType.Server)
}

const resetForm = () => {
  editServerID.value = undefined
  generalForm.value = {
    id: '',
    name: '',
    connectionString: '',
    parentId: '',
    colour: '',
  }
  previousParentId.value = ''
  generalFormRef.value?.restoreValidation()
  testing.value = false
  authPicker.value = 'password'
  nameWasEdited.value = false
  oidcConfig.value = {
    providerUrl: '',
    clientId: '',
    scopes: [],
    workloadIdentity: false,
    prompt: '',
    manualUrlMode: false,
  }
}

watch(
  () => dialogStore.dialogs[DialogType.Server].visible,
  async (visible: boolean) => {
    if (!visible) {
      return
    }
    resetForm()
    const data = dialogStore.serverDialogData
    if (data?.mode === 'edit' || data?.mode === 'clone') {
      editServerID.value = data.serverId
      const server = await serverStore.getServerDetails(data.serverId)
      if (server != null) {
        generalForm.value = {
          id: server.id,
          name: server.name,
          colour: server.colour,
          connectionString: server.uri,
          parentId: server.parentID || '',
        }
        previousParentId.value = generalForm.value.parentId
        authPicker.value = server.authMethod ?? 'password'
        if (server.oidcConfig) {
          oidcConfig.value = { ...server.oidcConfig }
        }
        nameWasEdited.value = true
      }
      return
    }
    if (data?.mode === 'new') {
      if (data.uri) {
        generalForm.value.connectionString = data.uri
      }
      if (data.name) {
        generalForm.value.name = data.name
        nameWasEdited.value = true
      }
      return
    }
    // legacy parentId payload (ServerPane right-click "New server in group")
    const rawData = dialogStore.dialogs[DialogType.Server].data as
      | Record<string, string>
      | undefined
    if (rawData?.parentId) {
      generalForm.value.parentId = rawData.parentId
      previousParentId.value = rawData.parentId
    }
  },
  { immediate: true },
)

watch(
  () => generalForm.value.connectionString,
  (uri) => {
    if (lastChangeSource.value === 'picker') {
      lastChangeSource.value = null
      return
    }
    if (!uri) {
      return
    }
    const detected = detectAuthFromUri(uri).authMethod
    if (detected !== 'password' && authPicker.value !== detected) {
      lastChangeSource.value = 'uri'
      authPicker.value = detected
      nextTick(() => { lastChangeSource.value = null })
    }
    if (!nameWasEdited.value) {
      const host = getUriHost(uri)
      if (host) {
        generalForm.value.name = host
      }
    }
  },
)

function onPanelMethodChange(next: AuthMethod): void {
  if (lastChangeSource.value === 'uri') {
    lastChangeSource.value = null
    authPicker.value = next
    return
  }
  authPicker.value = next
}

function onOidcConfigChange(next: OIDCConfig): void {
  oidcConfig.value = next
}

function onPanelUriChange(next: string): void {
  if (next === generalForm.value.connectionString) {
    return
  }
  lastChangeSource.value = 'picker'
  generalForm.value.connectionString = next
  nextTick(() => {
    lastChangeSource.value = null
  })
}

const onTestConnection = async () => {
  if (authPicker.value === 'oidc') {
    const notifier = useNotifier()
    notifier.info(i18n.t('serverPane.dialogs.server.testOIDCUnsupported'))
    return
  }

  testing.value = true

  let testingMessage = ''
  let testingDetail = ''
  try {
    const result = await connectionsProxy.TestConnection(generalForm.value.connectionString)
    if (!result.isSuccess) {
      testingMessage = i18n.t(`errors.${result.errorCode}`)
      testingDetail = result.errorDetail
    }
  } catch (e: unknown) {
    const err = e as Error
    testingMessage = err.message
  } finally {
    testing.value = false
  }

  const notifier = useNotifier()
  if (!isEmpty(testingMessage)) {
    notifier.error(testingMessage, {
      title: i18n.t('serverPane.dialogs.server.testFailure'),
      detail: testingDetail,
    })
  } else {
    notifier.success(i18n.t('serverPane.dialogs.server.testSuccess'))
  }
}
</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.Server].visible"
    :closable="false"
    :mask-closable="false"
    :on-after-leave="resetForm"
    :show-icon="false"
    :title="
      isEditMode
        ? $t('serverPane.dialogs.server.editTitle')
        : $t('serverPane.dialogs.server.newTitle')
    "
    close-on-esc
    preset="dialog"
    style="width: 600px"
    transform-origin="center"
    @esc="onClose">
    <n-spin :show="closingConnection">
      <n-tabs
        v-model:value="tab"
        animated
        pane-style="min-height: 50vh;"
        placement="left"
        tab-style="justify-content: right; font-weight: 420;"
        type="line">
        <n-tab-pane
          :tab="$t('serverPane.dialogs.server.generalTab')"
          display-directive="show:lazy"
          name="general">
          <n-form
            ref="generalFormRef"
            :model="generalForm"
            :rules="generalFormRules()"
            :show-require-mark="false"
            label-placement="top">
            <n-grid :x-gap="10">
              <n-form-item-gi
                :label="$t('serverPane.dialogs.server.connectionString')"
                :span="24"
                path="connectionString"
                required>
                <n-input
                  v-model:value="generalForm.connectionString"
                  :placeholder="$t('serverPane.dialogs.server.connectionStringTip')" />
              </n-form-item-gi>
              <n-form-item-gi
                :label="$t('serverPane.dialogs.server.name')"
                :span="24"
                path="name"
                required>
                <n-input
                  v-model:value="generalForm.name"
                  :placeholder="$t('serverPane.dialogs.server.nameTip')"
                  @input="nameWasEdited = true" />
              </n-form-item-gi>
              <n-form-item-gi :label="$t('serverPane.dialogs.server.group')" :span="24" required>
                <n-tree-select
                  v-model:value="generalForm.parentId"
                  :options="groupOptions"
                  key-field="id"
                  label-field="name"
                  @update:value="onParentChange" />
              </n-form-item-gi>
              <n-form-item-gi
                :label="$t('serverPane.dialogs.server.colour')"
                :span="24"
                path="colour">
                <div
                  v-for="colour in serverColors"
                  :key="colour"
                  :style="{
                    backgroundColor: colour,
                    borderColor:
                      generalForm.colour === colour
                        ? themeVars.textColorBase
                        : themeVars.borderColor,
                  }"
                  class="color-preset-item"
                  @click="generalForm.colour = colour">
                  <n-icon v-if="isEmpty(colour)" :component="XCircleIcon" size="24" />
                </div>
              </n-form-item-gi>
              <n-form-item-gi :span="24" :show-feedback="false">
                <n-text depth="3" style="font-size: 12px">
                  {{ $t('serverPane.dialogs.server.auth.hint', { mechanism: hintMechanismLabel }) }}
                </n-text>
              </n-form-item-gi>
            </n-grid>
          </n-form>
        </n-tab-pane>
        <n-tab-pane
          :tab="$t('serverPane.dialogs.server.authenticationTab')"
          display-directive="show:lazy"
          name="authentication">
          <AuthenticationPanel
            :uri="generalForm.connectionString"
            :method="authPicker"
            :oidc-config="oidcConfig"
            @update:uri="onPanelUriChange"
            @update:method="onPanelMethodChange"
            @update:oidc-config="onOidcConfigChange"
          />
        </n-tab-pane>
      </n-tabs>

    </n-spin>
    <template #action>
      <div class="flex-item-expand">
        <n-button
          :disabled="closingConnection || !generalForm.connectionString"
          :focusable="false"
          :loading="testing"
          @click="onTestConnection">
          {{ $t('serverPane.dialogs.server.test') }}
        </n-button>
      </div>
      <div class="flex-item n-dialog__action">
        <n-button :disabled="closingConnection" :focusable="false" @click="onClose">
          {{ $t('common.cancel') }}
        </n-button>
        <n-button
          :disabled="closingConnection || !generalForm.name || !generalForm.connectionString"
          :focusable="false"
          type="primary"
          @click="onSaveServer">
          {{ isEditMode ? $t('common.update') : $t('common.confirm') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style lang="scss" scoped>
.color-preset-item {
  width: 24px;
  height: 24px;
  margin-right: 2px;
  border-width: 3px;
  border-style: solid;
  cursor: pointer;
  border-radius: 50%;
}
</style>

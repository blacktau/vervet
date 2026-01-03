<script setup lang="ts">
import { type FormInst, type FormRules, type TreeSelectOption, useThemeVars } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { computed, nextTick, ref, watch } from 'vue'
import { every, includes, isEmpty } from 'lodash'
import { XCircleIcon } from '@heroicons/vue/24/outline'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { useMessager, useNotifier } from '@/utils/dialog.ts'
import { parseUri } from '@/features/server-pane/connectionStrings.ts'
import { filterGroupMap } from '@/features/server-pane/helpers.ts'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'

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
const showTestResult = ref<boolean>(false)
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
  if (isEmpty(editServerID.value)) {
    return false
  }
  return browserStore.isConnected(editServerID.value)
})

const onSaveServer = async () => {
  console.log('onSaveServer', generalFormRef.value)
  await generalFormRef.value?.validate((err) => {
    if (err) {
      nextTick(() => (tab.value = 'general'))
    }
  })

  const messager = useMessager()
  if (!isEditMode.value) {
    const result = await serverStore.saveServer(
      generalForm.value.name,
      generalForm.value.connectionString,
      generalForm.value.parentId,
    )
    if (!result.success) {
      messager.error(result.msg || 'unknown error')
      return
    }
  } else {
    console.log('editServerID', editServerID.value)

    const result = await serverStore.updateServer(
      editServerID.value || null,
      generalForm.value.name,
      generalForm.value.connectionString,
      generalForm.value.parentId,
      generalForm.value.colour
    )
    if (!result.success) {
      messager.error(result.msg || 'unknown error')
      return
    }
  }

  messager.success(i18n.t('common.dialog.handleSuccess'))
  onClose()
}

const groupOptions = computed(() => {
  const nodes = serverStore.serverTree
  const options: TreeSelectOption[] = []
  for (let i = 0, ln = nodes.length; i < ln; ++i) {
    const option = filterGroupMap(nodes[i]!)
    if (!!option) {
      options.push(option)
    }
  }
  options.splice(0, 0, {
    label: i18n.t('serverPane.dialogs.common.noGroup'),
    key: '',
  })
  return options
})

const onClose = () => {
  dialogStore.hide(DialogType.Server)
}

const resetForm = () => {
  generalForm.value = {
    id: '',
    name: '',
    connectionString: '',
    parentId: '',
    colour: '',
  }
  generalFormRef.value?.restoreValidation()
  testing.value = false
}

watch(
  () => dialogStore.dialogs[DialogType.Server].visible,
  async (visible: boolean) => {
    console.log('ServerDialog->watch.visible:', visible)
    if (visible) {
      resetForm()
      const data = dialogStore.serverDialogData
      console.log('dialog data', data)
      if (dialogStore.serverDialogData?.mode == 'edit') {
        editServerID.value = data?.serverId
        const server = await serverStore.getServerDetails(data?.serverId)
        if (server != null) {
          generalForm.value = {
            id: server.id,
            name: server.name,
            colour: server.colour,
            connectionString: server.uri,
            parentId: server.parentID || '',
          }
        }
      }
    }
  },
)

const onTestConnection = async () => {
  testing.value = true

  let testingResult = ''
  try {
    const result = await connectionsProxy.TestConnection(generalForm.value.connectionString)
    if (!result.isSuccess) {
      testingResult = result.error
    }
  } catch (e: unknown) {
    const err = e as Error
    testingResult = err.message
  } finally {
    testing.value = false
  }

  const notifier = useNotifier()
  if (!isEmpty(testingResult)) {
    notifier.error(testingResult, {
      title: i18n.t('serverPane.dialogs.server.testFailure'),
    })
  } else {
    notifier.success(i18n.t('serverPane.dialogs.server.testSuccess'))
  }
  showTestResult.value = true
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
                :label="$t('serverPane.dialogs.server.name')"
                :span="24"
                path="name"
                required>
                <n-input
                  v-model:value="generalForm.name"
                  :placeholder="$t('serverPane.dialogs.server.nameTip')" />
              </n-form-item-gi>
              <n-form-item-gi
                :label="$t('serverPane.dialogs.server.group')"
                :span="24"
                required>
                <n-tree-select :options="groupOptions" v-model:value="generalForm.parentId" />
              </n-form-item-gi>
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
                :label="$t('serverPane.dialogs.server.colour')"
                :span="24"
                path="colour">
                <div
                  v-for="colour in serverColors"
                  :key="colour"
                  :style="{
                    backgroundColor: colour,
                    borderColor:
                      generalForm.colour === colour ? themeVars.textColorBase : themeVars.borderColor,
                  }"
                  class="color-preset-item"
                  @click="generalForm.colour = colour">
                  <n-icon v-if="isEmpty(colour)" :component="XCircleIcon" size="24" />
                </div>
              </n-form-item-gi>
            </n-grid>
          </n-form>
        </n-tab-pane>
      </n-tabs>

      <!--      <n-alert-->
      <!--        v-if="showTestResult"-->
      <!--        :on-close="-->
      <!--          () => {-->
      <!--            testResult = ''-->
      <!--            showTestResult = false-->
      <!--          }-->
      <!--        "-->
      <!--        :title="isEmpty(testResult) ? '' : $t('dialog.server.testFailure')"-->
      <!--        :type="isEmpty(testResult) ? 'success' : 'error'"-->
      <!--        closable>-->
      <!--        <template v-if="isEmpty(testResult)">{{ $t('dialog.server.testSuccess') }}</template>-->
      <!--        <template v-else>{{ testResult }}</template>-->
      <!--      </n-alert>-->
    </n-spin>
    <template #action>
      <div class="flex-item-expand">
        <n-button
          :disabled="closingConnection"
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
          :disabled="closingConnection"
          :focusable="false"
          type="primary"
          @click="onSaveServer">
          {{ isEditMode ? $t('common.update') : $t('common.confirm') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped lang="scss">
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

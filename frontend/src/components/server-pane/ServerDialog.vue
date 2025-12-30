<script setup lang="ts">
import { type FormInst, type FormRules, type TreeSelectOption, useThemeVars } from 'naive-ui'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useDataBrowserStore } from '@/components/data-browser/browserStore.ts'
import { type RegisteredServerNode, useServerStore } from '@/components/server-pane/serverStore.ts'
import { useI18n } from 'vue-i18n'
import { computed, nextTick, ref, watch } from 'vue'
import { every, includes, isEmpty } from 'lodash'
import { useMessager, useNotifier } from '@/utils/dialog.ts'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import { parseUri } from '@/components/server-pane/connectionStrings.ts'

type EditableRegisteredServer = {
  id: string
  name: string
  connectionString: string
  parentId: string
}

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const serverStore = useServerStore()
const browserStore = useDataBrowserStore()
const i18n = useI18n()

const dialogState = dialogStore.serverDialogData

const editServerID = ref(dialogState?.serverId ?? '')
const tab = ref('general')
const showTestResult = ref<boolean>(false)

const generalForm = ref<EditableRegisteredServer>({
  id: '',
  name: '',
  parentId: '',
  connectionString: '',
})
const generalFormRef = ref<FormInst | undefined>(undefined)
const testing = ref(false)

const generalFormRules = () => {
  const requiredMsg = i18n.t('dialog.common.fieldRequired')
  const illegalChars = ['/', '\\']
  return {
    name: [
      { required: true, message: requiredMsg, trigger: 'input' },
      {
        validator: (rule, value) => {
          return every(illegalChars, (c) => !includes(value, c))
        },
        message: i18n.t('dialog.common.illegalCharacters'),
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

const isEditMode = computed(() => dialogState?.mode === 'edit')

const closingConnection = computed(() => {
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
    const result = await serverStore.updateServer(
      editServerID.value,
      generalForm.value.name,
      generalForm.value.connectionString,
      generalForm.value.parentId,
    )
    if (!result.success) {
      messager.error(result.msg || 'unknown error')
      return
    }
  }

  messager.success(i18n.t('dialog.common.handleSuccess'))
  onClose()
}

const mapNode = (node: RegisteredServerNode) => {
  if (!node.isGroup) {
    return undefined
  }

  const children: TreeSelectOption[] = []
  for (let i = 0, ln = node.children.length; i < ln; ++i) {
    if (!node.children[i]?.isGroup) {
      continue
    }

    const child = mapNode(node.children[i]!)
    if (child) {
      children.push(child)
    }
  }

  return {
    label: node.name,
    key: node.id,
    children: children,
  } as TreeSelectOption
}

const groupOptions = computed(() => {
  const nodes = serverStore.serverTree
  const options: TreeSelectOption[] = []
  for (let i = 0, ln = nodes.length; i < ln; ++i) {
    const option = mapNode(nodes[i]!)
    if (!!option) {
      options.push(option)
    }
  }
  options.splice(0, 0, {
    label: i18n.t('dialog.server.noGroup'),
    key: '',
  })
  return options
})

const onClose = () => {
  dialogStore.closeDialog(DialogType.Server)
}

const resetForm = () => {
  generalForm.value = {
    id: '',
    name: '',
    connectionString: '',
    parentId: '',
  }
  generalFormRef.value?.restoreValidation()
  testing.value = false
}

watch(
  () => dialogStore.dialogs[DialogType.Server]?.visible ?? false,
  (visible: boolean) => {
    if (visible) {
      resetForm()
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
      title: i18n.t('dialog.server.testFailure'),
    })
  } else {
    notifier.success(i18n.t('dialog.server.testSuccess'))
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
    :title="isEditMode ? $t('dialog.server.editTitle') : $t('dialog.server.newTitle')"
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
          :tab="$t('dialog.server.generalTab')"
          display-directive="show:lazy"
          name="general">
          <n-form
            ref="generalFormRef"
            :model="generalForm"
            :rules="generalFormRules()"
            :show-require-mark="false"
            label-placement="top">
            <n-grid :x-gap="10">
              <n-form-item-gi :label="$t('dialog.server.name')" :span="24" path="name" required>
                <n-input
                  v-model:value="generalForm.name"
                  :placeholder="$t('dialog.server.nameTip')" />
              </n-form-item-gi>
              <n-form-item-gi
                v-if="!isEditMode"
                :label="$t('dialog.server.group')"
                :span="24"
                required>
                <n-tree-select :options="groupOptions" v-model:value="generalForm.parentId" />
              </n-form-item-gi>
              <n-form-item-gi
                :label="$t('dialog.server.connectionString')"
                :span="24"
                path="connectionString"
                required>
                <n-input
                  v-model:value="generalForm.connectionString"
                  :placeholder="$t('dialog.server.connectionStringTip')" />
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
          {{ $t('dialog.server.test') }}
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
          {{ isEditMode ? $t('settings.general.update') : $t('common.confirm') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped lang="scss"></style>

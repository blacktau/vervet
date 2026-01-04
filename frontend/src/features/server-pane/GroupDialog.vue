<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, reactive, ref, watchEffect } from 'vue'
import { every, get, includes, isEmpty } from 'lodash'
import { type FormItemRule, type FormInst, type TreeSelectOption } from 'naive-ui'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { type RegisteredServerNode, useServerStore } from '@/features/server-pane/serverStore.ts'
import { useMessager } from '@/utils/dialog.ts'
import { filterGroupMap } from '@/features/server-pane/helpers.ts'

const dialogStore = useDialogStore()
const serverStore = useServerStore()
const i18n = useI18n()

const editGroup = ref('')
const groupForm = reactive<{
  name: string
  parentId?: string
}>({
  name: '',
})

const groupFormRef = ref<FormInst | null>(null)

const formRules = computed(() => {
  const requiredMsg = i18n.t('common.dialog.fieldRequired')
  const illegalChars = ['/', '\\']
  return {
    name: [
      { required: true, message: requiredMsg, trigger: 'input' },
      {
        validator: (rule: FormItemRule, value: string) => {
          return every(illegalChars, (c) => !includes(value, c))
        },
        message: i18n.t('common.dialog.illegalCharacters'),
        trigger: 'input',
      },
    ],
  }
})

const isEditMode = computed(
  () => ((dialogStore.dialogs[DialogType.Group].data as string) || '').length > 0,
)

const onConfirm = async () => {
  const messager = useMessager()
  try {
    await groupFormRef.value?.validate((errs) => {
      const err = get(errs, '0.0.message')
      if (err != null) {
        const messager = useMessager()
        messager.error(err)
      }
    })

    const { name, parentId } = groupForm

    if (isEditMode.value) {
      const { success, msg } = await serverStore.updateGroup(editGroup.value, name, parentId)

      if (success) {
        messager.success(i18n.t('common.dialog.handleSuccess'))
      } else {
        messager.error(msg!)
      }
    } else {
      const { success, msg } = await serverStore.createGroup(name)
      if (success) {
        messager.success(i18n.t('common.dialog.handleSuccess'))
      } else {
        messager.error(msg!)
      }
    }
  } catch (e) {
    const err = e as Error
    messager.error(err.message)
  }
}

const onClose = () => {
  if (isEditMode.value) {
    dialogStore.closeNewGroupDialog()
  } else {
    dialogStore.closeRenameGroupDialog()
  }
}

const groupOptions = computed(() => {
  const nodes = serverStore.serverTree
  const options: RegisteredServerNode[] = []
  for (let i = 0, ln = nodes.length; i < ln; ++i) {
    const option = filterGroupMap(nodes[i]!)
    if (option?.id == editGroup.value) {
      continue
    }

    if (!!option) {
      options.push(option)
    }
  }
  options.splice(0, 0, {
    name: i18n.t('serverPane.dialogs.common.noGroup'),
    id: '',
    isGroup: true,
    isCluster: false,
    isSrv: false,
    children: [],
    colour: '',
  })
  return options
})

watchEffect(() => {
  if (dialogStore.dialogs[DialogType.Group].visible) {
    const groupId = dialogStore.dialogs[DialogType.Group].data as string
    const group = serverStore.findServerById(groupId)
    editGroup.value = groupId
    groupForm.name = group?.name || ''
    groupForm.parentId = group?.parentID
  }
})
</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.Group].visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium' }"
    :negative-text="$t('common.cancel')"
    :positive-button-props="{ size: 'medium' }"
    :positive-text="$t('common.confirm')"
    :show-icon="false"
    :title="isEditMode ? $t('serverPane.dialogs.group.edit') : $t('serverPane.dialogs.group.new')"
    close-on-esc
    preset="dialog"
    transform-origin="center"
    @esc="onClose"
    @positive-click="onConfirm"
    @negative-click="onClose">
    <n-form
      ref="groupFormRef"
      :model="groupForm"
      :rules="formRules"
      :show-label="false"
      :show-require-mark="false"
      label-placement="top">
      <n-form-item :label="$t('serverPane.dialogs.group.name')" path="name" required>
        <n-input
          v-model:value="groupForm.name"
          :placeholder="$t('serverPane.dialogs.group.namePlaceholder')" />
      </n-form-item>
      <n-form-item :label="$t('serverPane.dialogs.group.parent')" :span="24" required>
        <n-tree-select
          :options="groupOptions"
          v-model:value="groupForm.parentId"
          key-field="id"
          label-field="name" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>

<style scoped lang="scss"></style>

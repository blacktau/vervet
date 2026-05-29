<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { computed, reactive, ref, watch, watchEffect } from 'vue'
import { every, get, includes } from 'lodash'
import { type FormInst, type FormItemRule } from 'naive-ui'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { type RegisteredServerNode, useServerStore } from '@/features/server-pane/serverStore.ts'
import { useMessager } from '@/utils/dialog.ts'
import { filterGroupMap } from '@/features/server-pane/helpers.ts'
import { asCreatePayload, isEditPayload } from '@/features/server-pane/groupDialog.ts'

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

function siblingGroupsAt(parentId: string): RegisteredServerNode[] {
  if (!parentId) {
    return serverStore.serverTree.filter((n) => n.isGroup)
  }
  const parent = serverStore.findServerById(parentId)
  return (parent?.children || []).filter((n) => n.isGroup)
}

function isDuplicateSibling(name: string, parentId: string, excludeId: string): boolean {
  const target = name.trim().toLowerCase()
  if (!target) {
    return false
  }
  return siblingGroupsAt(parentId).some(
    (n) => n.id !== excludeId && n.name.trim().toLowerCase() === target,
  )
}

const formRules = computed(() => {
  const requiredMsg = i18n.t('common.dialog.fieldRequired')
  const illegalChars = ['/', '\\']
  return {
    name: [
      { required: true, message: requiredMsg, trigger: ['input', 'blur'] },
      {
        validator: (rule: FormItemRule, value: string) => {
          return every(illegalChars, (c) => !includes(value, c))
        },
        message: i18n.t('common.dialog.illegalCharacters'),
        trigger: ['input', 'blur'],
      },
      {
        validator: (rule: FormItemRule, value: string) => {
          return !isDuplicateSibling(value, groupForm.parentId || '', editGroup.value)
        },
        message: i18n.t('errors.duplicate_group_name'),
        trigger: ['input', 'blur'],
      },
    ],
  }
})

const isEditMode = computed(() =>
  isEditPayload(dialogStore.dialogs[DialogType.Group].data),
)

const onConfirm = async (): Promise<boolean> => {
  const messager = useMessager()
  try {
    let validationError = false
    await groupFormRef.value?.validate((errs) => {
      const err = get(errs, '0.0.message')
      if (err != null) {
        validationError = true
        messager.error(err)
      }
    })

    if (validationError) {
      return false
    }

    const { name, parentId } = groupForm

    if (isEditMode.value) {
      const { success, msg } = await serverStore.updateGroup(editGroup.value, name, parentId)

      if (success) {
        messager.success(i18n.t('common.dialog.handleSuccess'))
        onClose()
        return true
      }
      messager.error(msg!)
      return false
    }

    const result = await serverStore.createGroup(name, parentId)
    if (result.success) {
      const payload = asCreatePayload(dialogStore.dialogs[DialogType.Group].data)
      payload?.onCreated?.(result.id)
      messager.success(i18n.t('common.dialog.handleSuccess'))
      onClose()
      return true
    }
    messager.error(result.msg!)
    return false
  } catch (e) {
    const err = e as Error
    messager.error(err.message)
    return false
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

watch(
  () => groupForm.parentId,
  () => {
    if (!dialogStore.dialogs[DialogType.Group].visible) {
      return
    }
    groupFormRef.value?.validate(undefined, (rule) => rule?.key === 'name').catch(() => {})
  },
)

watchEffect(() => {
  if (!dialogStore.dialogs[DialogType.Group].visible) {
    return
  }
  const rawData = dialogStore.dialogs[DialogType.Group].data
  if (isEditPayload(rawData)) {
    const group = serverStore.findServerById(rawData)
    editGroup.value = rawData
    groupForm.name = group?.name || ''
    groupForm.parentId = group?.parentID || ''
    return
  }
  const payload = asCreatePayload(rawData)
  editGroup.value = ''
  groupForm.name = ''
  groupForm.parentId = payload?.parentId || ''
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
          :placeholder="$t('serverPane.dialogs.group.namePlaceholder')"
          @keyup.enter="onConfirm" />
      </n-form-item>
      <n-form-item :label="$t('serverPane.dialogs.group.parent')" :span="24" required>
        <n-tree-select
          v-model:value="groupForm.parentId"
          :options="groupOptions"
          key-field="id"
          label-field="name" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>

<style lang="scss" scoped></style>

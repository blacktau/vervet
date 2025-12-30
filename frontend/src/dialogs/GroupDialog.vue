<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { computed, reactive, ref, watchEffect } from 'vue'
import { every, get, includes, isEmpty } from 'lodash'
import { type FormItemRule, type FormInst } from 'naive-ui'
import { useDialogStore } from '@/stores/dialog.ts'
import { useServerStore } from '@/components/server-pane/serverStore.ts'
import { useMessager } from '@/utils/dialog.ts'

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
  const requiredMsg = i18n.t('dialog.requiredField')
  const illegalChars = ['/', '\\']
  return {
    name: [
      { required: true, message: requiredMsg, trigger: 'input' },
      {
        validator: (rule: FormItemRule, value: string) => {
          return every(illegalChars, (c) => !includes(value, c))
        },
        message: i18n.t('dialog.illegalCharacters'),
        trigger: 'input',
      },
    ],
  }
})

const isRenameMode = computed(() => !isEmpty(editGroup.value))

const dialogStore = useDialogStore()
const serverStore = useServerStore()
watchEffect(() => {
  if (dialogStore.groupDialogVisible) {
    groupForm.name = editGroup.value = dialogStore.editGroup
  }
})

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

    if (isRenameMode.value) {
      const { success, msg } = await serverStore.renameGroup(editGroup.value, name)

      if (success) {
        messager.success(i18n.t('dialog.handleSuccess'))
      } else {
        messager.error(msg!)
      }
    } else {
      const { success, msg } = await serverStore.createGroup(name)
      if (success) {
        messager.success(i18n.t('dialog.handleSuccess'))
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
  if (isRenameMode.value) {
    dialogStore.closeNewGroupDialog()
  } else {
    dialogStore.closeRenameGroupDialog()
  }
}

</script>

<template>
  <n-modal v-model:show="dialogStore.groupDialogVisible"
           :closable="false"
           :mask-closable="false"
           :negative-button-props="{ size: 'medium' }"
           :negative-text="$t('common.cancel')"
           :positive-button-props="{ size: 'medium' }"
           :positive-text="$t('common.confirm')"
           :show-icon="false"
           :title="isRenameMode ? $t('dialog.group.rename') : $t('dialog.group.new')"
           close-on-esc
           preset="dialog"
           transform-origin="center"
           @esc="onClose"
           @positive-click="onConfirm"
           @negative-click="onClose">
    <n-form ref="groupFormRef"
            :model="groupForm"
            :rules="formRules"
            :show-label="false"
            :show-require-mark="false"
            label-placement="top">
      <n-form-item :label="$t('dialog.group.name')" path="name" required>
        <n-input v-model:value="groupForm.name" placeholder="" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>

<style scoped lang="scss"></style>

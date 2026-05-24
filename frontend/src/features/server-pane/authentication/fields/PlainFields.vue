<template>
  <n-form label-placement="top">
    <n-form-item
      :label="$t('serverPane.dialogs.server.auth.plain.username')"
      :validation-status="usernameTouched && !fields.username ? 'warning' : undefined"
      :feedback="usernameTouched && !fields.username ? $t('serverPane.dialogs.server.auth.warnings.missingUsername') : ''">
      <n-input
        :value="fields.username"
        data-testid="plain-username"
        @update:value="(v: string) => update({ username: v })"
        @blur="usernameTouched = true"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.plain.password')">
      <n-input
        :value="fields.password"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ password: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.plain.authSource')">
      <n-input
        :value="fields.authSource ?? '$external'"
        :placeholder="$t('serverPane.dialogs.server.auth.plain.authSourcePlaceholder')"
        @update:value="(v: string) => update({ authSource: v || undefined })"
      />
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { parsePlain, serialisePlain, type PlainAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const fields = computed<PlainAuth>(() => parsePlain(props.uri))
const usernameTouched = ref(false)

function update(patch: Partial<PlainAuth>): void {
  emit('update:uri', serialisePlain(props.uri, { ...fields.value, ...patch }))
}
</script>

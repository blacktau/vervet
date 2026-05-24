<template>
  <n-form label-placement="top">
    <n-form-item
      :label="$t('serverPane.dialogs.server.auth.aws.accessKeyId')"
      :validation-status="fields.accessKeyId ? undefined : 'warning'"
      :feedback="fields.accessKeyId ? '' : $t('serverPane.dialogs.server.auth.warnings.missingAwsKey')">
      <n-input
        :value="fields.accessKeyId"
        data-testid="aws-access-key"
        @update:value="(v: string) => update({ accessKeyId: v })"
      />
    </n-form-item>
    <n-form-item
      :label="$t('serverPane.dialogs.server.auth.aws.secretAccessKey')"
      :validation-status="fields.secretAccessKey ? undefined : 'warning'"
      :feedback="fields.secretAccessKey ? '' : $t('serverPane.dialogs.server.auth.warnings.missingAwsSecret')">
      <n-input
        :value="fields.secretAccessKey"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ secretAccessKey: v })"
      />
    </n-form-item>
    <n-form-item :label="$t('serverPane.dialogs.server.auth.aws.sessionToken')">
      <n-input
        :value="fields.sessionToken ?? ''"
        type="password"
        show-password-on="click"
        data-testid="aws-session-token"
        @update:value="(v: string) => update({ sessionToken: v || undefined })"
      />
      <template #feedback>
        {{ $t('serverPane.dialogs.server.auth.aws.sessionTokenHelp') }}
      </template>
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { parseAws, serialiseAws, type AwsAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const fields = computed<AwsAuth>(() => parseAws(props.uri))

function update(patch: Partial<AwsAuth>): void {
  emit('update:uri', serialiseAws(props.uri, { ...fields.value, ...patch }))
}
</script>

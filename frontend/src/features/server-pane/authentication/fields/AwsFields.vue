<template>
  <n-form label-placement="top">
    <n-form-item
      :label="$t('serverPane.dialogs.server.auth.aws.accessKeyId')"
      :validation-status="keyTouched && !fields.accessKeyId ? 'warning' : undefined"
      :feedback="keyTouched && !fields.accessKeyId ? $t('serverPane.dialogs.server.auth.warnings.missingAwsKey') : ''">
      <n-input
        :value="fields.accessKeyId"
        data-testid="aws-access-key"
        @update:value="(v: string) => update({ accessKeyId: v })"
        @blur="keyTouched = true"
      />
    </n-form-item>
    <n-form-item
      :label="$t('serverPane.dialogs.server.auth.aws.secretAccessKey')"
      :validation-status="secretTouched && !fields.secretAccessKey ? 'warning' : undefined"
      :feedback="secretTouched && !fields.secretAccessKey ? $t('serverPane.dialogs.server.auth.warnings.missingAwsSecret') : ''">
      <n-input
        :value="fields.secretAccessKey"
        type="password"
        show-password-on="click"
        @update:value="(v: string) => update({ secretAccessKey: v })"
        @blur="secretTouched = true"
      />
    </n-form-item>
    <n-form-item>
      <template #label>
        <span class="label-with-help">
          {{ $t('serverPane.dialogs.server.auth.aws.sessionToken') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon :component="InformationCircleIcon" size="14" />
            </template>
            {{ $t('serverPane.dialogs.server.auth.aws.sessionTokenHelp') }}
          </n-tooltip>
        </span>
      </template>
      <n-input
        :value="fields.sessionToken ?? ''"
        type="password"
        show-password-on="click"
        data-testid="aws-session-token"
        @update:value="(v: string) => update({ sessionToken: v || undefined })"
      />
    </n-form-item>
  </n-form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import { parseAws, serialiseAws, type AwsAuth } from '../authUri'

const props = defineProps<{ uri: string }>()
const emit = defineEmits<{ (e: 'update:uri', value: string): void }>()

const fields = computed<AwsAuth>(() => parseAws(props.uri))
const keyTouched = ref(false)
const secretTouched = ref(false)

function update(patch: Partial<AwsAuth>): void {
  emit('update:uri', serialiseAws(props.uri, { ...fields.value, ...patch }))
}
</script>

<style scoped>
.label-with-help {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}
</style>

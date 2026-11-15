<script lang="ts" setup>
import { computed } from 'vue'
import { useUpdateStore } from './updateStore'

/**
 * The buttons for an available update. A copy that can install updates itself
 * offers "Update now" and then "Restart now"; any other copy links to the
 * release page.
 */
const props = defineProps<{
  /** Text buttons and no release-page link, for the notification. */
  compact?: boolean
}>()

const emit = defineEmits<{
  /** The user dismissed the update, so a notification can close. */
  (e: 'dismissed'): void
}>()

const updates = useUpdateStore()

const busy = computed(() => updates.phase === 'downloading' || updates.phase === 'verifying')
const showInstall = computed(
  () => updates.canInstall && (updates.phase === 'idle' || updates.phase === 'error'),
)
// After a failed install the release page is the way out, e.g. for a release
// with no signed build for this platform.
const showRelease = computed(
  () =>
    !updates.canInstall ||
    updates.phase === 'error' ||
    (!props.compact && updates.phase !== 'ready'),
)
const showDismiss = computed(() => updates.phase === 'idle' || updates.phase === 'error')

async function dismiss() {
  await updates.dismiss()
  emit('dismissed')
}
</script>

<template>
  <div v-if="!busy" class="update-actions">
    <n-button
      v-if="updates.phase === 'ready'"
      :text="compact"
      type="primary"
      @click="updates.restart">
      {{ $t('settings.updates.restartNow') }}
    </n-button>
    <n-button v-if="showInstall" :text="compact" type="primary" @click="updates.install">
      {{
        updates.phase === 'error'
          ? $t('settings.updates.tryAgain')
          : $t('settings.updates.updateNow')
      }}
    </n-button>
    <n-button
      v-if="showRelease"
      :text="compact"
      :type="updates.canInstall ? 'default' : 'primary'"
      @click="updates.openReleasePage">
      {{ $t('settings.updates.viewRelease') }}
    </n-button>
    <n-button v-if="showDismiss" :text="compact" @click="dismiss">
      {{ $t('settings.updates.dismiss') }}
    </n-button>
  </div>
</template>

<style scoped>
.update-actions {
  display: flex;
  gap: 8px;
}
</style>

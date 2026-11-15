<script lang="ts" setup>
import { computed } from 'vue'
import { useUpdateStore } from './updateStore'

/** How far an in-app update has got: download progress, verifying, ready, or the error. */
const updates = useUpdateStore()

const downloading = computed(() => updates.phase === 'downloading')
</script>

<template>
  <div v-if="updates.phase !== 'idle'" class="update-status">
    <template v-if="downloading">
      <span>{{ $t('settings.updates.downloading') }}</span>
      <n-progress
        type="line"
        :percentage="updates.progress ?? 0"
        :show-indicator="updates.progress !== null"
        :processing="updates.progress === null" />
    </template>
    <span v-else-if="updates.phase === 'verifying'">{{ $t('settings.updates.verifying') }}</span>
    <span v-else-if="updates.phase === 'ready'">{{ $t('settings.updates.ready') }}</span>
    <n-text v-else-if="updates.phase === 'error'" type="error">
      {{ $t('settings.updates.installFailed', { error: updates.installError }) }}
    </n-text>
  </div>
</template>

<style scoped>
.update-status {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 240px;
}
</style>

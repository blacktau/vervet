<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { computed } from 'vue'
import * as runtime from 'wailsjs/runtime/runtime'
import WindowMin from '@/features/icon/WindowMin.vue'
import WindowRestore from '@/features/icon/WindowRestore.vue'
import WindowMax from '@/features/icon/WindowMax.vue'
import WindowClose from '@/features/icon/WindowClose.vue'

const themeVars = useThemeVars()
const props = withDefaults(defineProps<{
  size?: number
  maximised?: boolean
}>(), {
  size: 35,
})

const buttonSize = computed(() => {
  return props.size + 'px'
})

const handleMinimise = () => {
  runtime.WindowMinimise()
}

const handleMaximise = () => {
  runtime.WindowToggleMaximise()
}

const handleClose = () => {
  runtime.Quit()
}

</script>

<template>
  <n-space :size="0" :wrap-item="false" align="center" justify="center">
    <n-tooltip :delay="1000" :show-arrow="false">
      {{ $t('menu.minimise') }}
      <template #trigger>
        <div class="btn-wrapper" @click="handleMinimise">
          <window-min />
        </div>
      </template>
    </n-tooltip>
    <n-tooltip v-if="maximised" :delay="1000" :show-arrow="false">
      {{ $t('menu.restore') }}
      <template #trigger>
        <div class="btn-wrapper" @click="handleMaximise">
          <window-restore />
        </div>
      </template>
    </n-tooltip>
    <n-tooltip v-else :delay="1000" :show-arrow="false">
      {{ $t('menu.maximise') }}
      <template #trigger>
        <div class="btn-wrapper" @click="handleMaximise">
          <window-max />
        </div>
      </template>
    </n-tooltip>
    <n-tooltip :delay="1000" :show-arrow="false">
      {{ $t('menu.close') }}
      <template #trigger>
        <div class="btn-wrapper" @click="handleClose">
          <window-close />
        </div>
      </template>
    </n-tooltip>
  </n-space>
</template>

<style scoped lang="scss">
.btn-wrapper {
  width: v-bind('buttonSize');
  height: v-bind('buttonSize');
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  --wails-draggable: none;

  &:hover {
    cursor: pointer;
  }

  &:not(:last-child) {
    &:hover {
      background-color: v-bind('themeVars.closeColorHover');
    }

    &:active {
      background-color: v-bind('themeVars.closeColorPressed');
    }
  }

  &:last-child {
    &:hover {
      background-color: v-bind('themeVars.primaryColorHover');
    }

    &:active {
      background-color: v-bind('themeVars.primaryColorPressed');
    }
  }
}
</style>

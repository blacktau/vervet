<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { ref } from 'vue'

interface Props {
  size?: number
  minSize?: number
  maxSize?: number
  offset?: number
  disabled?: boolean
  borderWidth?: number
}

const themeVars = useThemeVars()

const props = withDefaults(defineProps<Props>(), {
  size: 300,
  minSize: 100,
  offset: 0,
  disabled: false,
  borderWidth: 4,
})

const emit = defineEmits<{
  (e: 'update:size', val: number): void
}>()

const resizing = ref(false)
const hover = ref(false)

const handleResize = (evt: MouseEvent) => {
  if (resizing.value) {
    let size = evt.clientY - props.offset
    if (size < props.minSize) {
      size = props.minSize
    }
    if (props.maxSize && props.maxSize > 0 && size > props.maxSize) {
      size = props.maxSize
    }
    emit('update:size', size)
  }
}

const stopResize = () => {
  resizing.value = false
  document.removeEventListener('mousemove', handleResize)
  document.removeEventListener('mouseup', stopResize)
}

const startResize = () => {
  if (props.disabled) {
    return
  }
  resizing.value = true
  document.addEventListener('mousemove', handleResize)
  document.addEventListener('mouseup', stopResize)
}

const handleMouseOver = () => {
  if (props.disabled) {
    return
  }
  hover.value = true
}
</script>

<template>
  <div :style="{ height: props.size + 'px' }" class="resizeable-wrapper-v flex-box-v">
    <slot></slot>
    <div
      :class="{
        'resize-divider-hover': hover,
        'resize-divider-drag': resizing,
        dragging: hover || resizing,
      }"
      :style="{
        height: props.borderWidth + 'px',
        bottom: Math.floor(-props.borderWidth / 2) + 'px',
      }"
      class="resize-divider"
      @mousedown="startResize"
      @mouseover="handleMouseOver"
      @mouseout="hover = false" />
  </div>
</template>

<style scoped lang="scss">
.resizeable-wrapper-v {
  position: relative;
  flex-shrink: 0;

  .resize-divider {
    position: absolute;
    left: 0;
    right: 0;
    transition: background-color 0.3s ease-in;
    z-index: 1;
  }

  .resize-divider-hover {
    background-color: v-bind('themeVars.borderColor');
  }

  .resize-divider-drag {
    background-color: v-bind('themeVars.primaryColor');
  }

  .dragging {
    cursor: row-resize !important;
  }
}
</style>

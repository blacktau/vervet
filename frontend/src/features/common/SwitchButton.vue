<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    value?: number
    size?: string
    icons?: string[]
    tTooltips?: string[]
    tTooltipPlacement?: string
    iconSize?: number | string
    color?: string
    stokeWidth?: number | string
    unselectedStrokeWidth?: number | string
  }>(),
  {
    value: 0,
    size: 'small',
    tTooltipPlacement: 'bottom',
    iconSize: 20,
    color: 'currentColor',
    stokeWidth: 3,
    unselectedStrokeWidth: 3,
  },
)

const emit = defineEmits<{
  (e: 'update:value', val?: number): void
}>()

const handleSwitch = (idx: number) => {
  if (idx !== props.value) {
    emit('update:value', idx)
  }
}
</script>

<template>
  <n-button-group>
    <n-tooltip v-for="(icon, i) in props.icons"
               :key="i"
               :disabled="!(props.tTooltips && props.tTooltips[i])"
               :placement="props.tTooltipPlacement"
               :show-arrow="false">
      <template #trigger>
        <n-button :focusable="false" :size="props.size" :tertiary="i !== props.value" @click="handleSwitch(i)">
          <template #icon>
            <n-icon :size="props.iconSize">
              <component
                :is="icon"
                :stroke-width="i !== props.value ? props.unselectedStrokeWidth : props.stokeWidth" />
            </n-icon>
          </template>
        </n-button>
      </template>
      {{ props.tTooltips ? $t(props.tTooltips[i]) : '' }}
    </n-tooltip>
  </n-button-group>
</template>

<style scoped lang="scss"></style>

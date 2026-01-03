<script setup lang="ts">
import {
  computed,
  type FunctionalComponent,
  type HTMLAttributes,
  useSlots,
  type VNode,
  type VNodeProps,
} from 'vue'

interface Props {
  tooltip?: string
  tTooltip?: string
  tooltipDelay?: number
  type?: string
  icon?: string | object | FunctionalComponent<HTMLAttributes & VNodeProps>
  size?: number | string
  color?: string
  strokeWidth?: number | string
  loading?: boolean
  border?: boolean
  disabled?: boolean
  buttonStyle?: string | object
  buttonClass?: string | object
  small?: boolean
  secondary?: boolean
  tertiary?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  tooltipDelay: 800,
  size: 20,
  color: '',
  strokeWidth: 3,
})

const emit = defineEmits<{
  (e: 'click'): void
}>()

const slots = useSlots()

const hasTooltip = computed(() => {
  return props.tooltip || props.tTooltip || slots.tooltip
})
</script>

<template>
  <n-tooltip
    v-if="hasTooltip"
    :delay="props.tooltipDelay"
    :keep-alive-on-hover="false"
    :show-arrow="false">
    <template #trigger>
      <n-button
        :class="props.buttonClass"
        :color="props.color"
        :disabled="props.disabled"
        :focusable="false"
        :loading="loading"
        :secondary="props.secondary"
        :size="props.small ? 'small' : ''"
        :style="props.buttonStyle"
        :tertiary="props.tertiary"
        :text="!props.border"
        :type="props.type"
        @click.prevent="emit('click')">
        <template #icon>
          <slot>
            <n-icon :color="props.color || 'currentColor'" :size="props.size">
              <component :is="icon" :stroke-width="props.strokeWidth" />
            </n-icon>
          </slot>
        </template>
      </n-button>
    </template>
    <slot name="tooltip">
      {{ props.tTooltip ? $t(props.tTooltip) : props.tooltip }}
    </slot>
  </n-tooltip>
  <n-button
    v-else
    :class="props.buttonClass"
    :color="props.color"
    :disabled="props.disabled"
    :focusable="false"
    :loading="loading"
    :secondary="props.secondary"
    :size="props.small ? 'small' : ''"
    :style="props.buttonStyle"
    :tertiary="props.tertiary"
    :text="!props.border"
    :type="props.type"
    @click.prevent="emit('click')">
    <template #icon>
      <slot>
        <n-icon :color="props.color || 'currentColor'" :size="props.size">
          <component :is="props.icon" :stroke-width="props.strokeWidth" />
        </n-icon>
      </slot>
    </template>
  </n-button>
</template>

<style scoped lang="scss"></style>

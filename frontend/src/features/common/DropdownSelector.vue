<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRender } from '@/utils/render'
import { computed, h, ref, Component } from 'vue'
import { DropdownOption, NText } from 'naive-ui'
import { isEmpty, some } from 'lodash'

interface MenuOption {
  key: string
  label: string
}

interface Props {
  value?: string
  options?: string[] | string[][]
  menuOptions?: MenuOption[]
  tooltip?: string
  icon?: string | Component
  default?: string
  disabled?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  value: '',
  options: () => [],
  menuOptions: () => [],
})

const emit = defineEmits<{
  (e: 'update:value', value: string): void
  (e: 'menu', key: string): void
}>()

const i18n = useI18n()
const render = useRender()

const renderHeader = () => {
  return h('div', { class: 'type-selector-header' }, [h(NText, null, () => props.tooltip)])
}

const dropdownOptions = computed(() => {
  const options: DropdownOption[] = []

  if (props.tooltip) {
    options.push(
      {
        key: 'header',
        type: 'render',
        render: renderHeader,
      },
      {
        key: 'header-divider',
        type: 'divider',
      }
    )
  }

  const isGrouped = props.options.some(Array.isArray)

  if (isGrouped) {
    const groupedOptions = props.options as string[][]
    for (let i = 0, ln = groupedOptions.length; i < ln; i++) {
      if (i !== 0 && !isEmpty(groupedOptions[i])) {
        options.push({
          key: 'header-divider' + (i + 1),
          type: 'divider',
        })
      }
      for (const option of groupedOptions[i]) {
        options.push({
          key: option,
          label: option,
        })
      }
    }
  } else {
    const singleOptions = props.options as string[]
    for (const option of singleOptions) {
      options.push({
        key: option,
        label: option,
      })
    }
  }

  if (!isEmpty(props.menuOptions)) {
    options.push({
      key: 'menu-divider',
      type: 'divider',
    })
    for (const { key, label } of props.menuOptions) {
      options.push({
        key,
        label: i18n.t(label),
      })
    }
  }
  return options
})

const onDropdownSelect = (key: string) => {
  if (some(props.menuOptions, { key })) {
    emit('menu', key)
  } else {
    emit('update:value', key)
  }
}

const showDropdown = ref(false)
const onDropdownShow = (show: boolean) => {
  showDropdown.value = show
}
</script>

<template>
  <n-dropdown
    :disabled="props.disabled"
    :options="dropdownOptions"
    :render-label="({ label }:DropdownOption) => render.renderLabel(label as string, { class: 'type-selector-item' })"
    :show-arrow="true"
    :value="props.value"
    trigger="click"
    @select="onDropdownSelect"
    @update:show="onDropdownShow">
    <n-tooltip :disabled="showDropdown" :show-arrow="false">
      {{ props.tooltip }}
      <template #trigger>
        <n-button :disabled="disabled" :focusable="false" quaternary>
          <template #icon>
            <n-icon>
              <component :is="icon" />
            </n-icon>
          </template>
        </n-button>
      </template>
    </n-tooltip>
  </n-dropdown>
</template>

<style scoped lang="scss">
.type-selector-header {
  height: 30px;
  line-height: 30px;
  font-size: 15px;
  font-weight: bold;
  text-align: center;
  padding: 0 10px;
}

.type-selector-item {
  min-width: 100px;
  text-align: center;
}
</style>

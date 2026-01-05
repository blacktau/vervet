<script setup lang="ts">
import { computed, nextTick, reactive } from 'vue'
import { debounce, isEmpty, trim } from 'lodash'
import { EqualsIcon } from '@heroicons/vue/24/outline'
import IconButton from '@/features/common/IconButton.vue'

const props = withDefaults(
  defineProps<{
    fullSearchIcon?: string | object
    debounceDelayMs?: number
    small?: boolean
    useGlob?: boolean
    exact?: boolean
  }>(),
  {
    debounceDelayMs: 500,
    small: false,
    useGlob: false,
    exact: false,
  },
)

const emit = defineEmits<{
  (e: 'filterChanged', filter: string, exact: boolean): void
  (e: 'matchChanged', match: string, filter: string, exact: boolean): void
  (e: 'exactChanged'): void
}>()

const inputData = reactive<{
  match: string
  filter: string
  exact: false
}>({
  match: '',
  filter: '',
  exact: false,
})

const hasMatch = computed(() => {
  return !isEmpty(trim(inputData.match))
})

const hasFilter = computed(() => {
  return !isEmpty(trim(inputData.filter))
})

const onExactChecked = () => {
  if (hasMatch.value) {
    nextTick(() => onForceFullSearch())
  }
}

const onFullSearch = () => {
  inputData.filter = trim(inputData.filter)
  if (!isEmpty(inputData.filter)) {
    inputData.match = inputData.filter
    inputData.filter = ''
    emit('matchChanged', inputData.match, inputData.filter, inputData.exact)
  }
}

const onForceFullSearch = () => {
  inputData.filter = trim(inputData.filter)
  emit('matchChanged', inputData.match, inputData.filter, inputData.exact)
}

const onInput = () => {
  debounce(() => emit('filterChanged', inputData.filter, inputData.exact), props.debounceDelayMs, {
    leading: true,
    trailing: true,
  })
}

const onClearFilter = () => {
  inputData.filter = ''
  onClearMatch()
}

const onUpdateMatch = () => {
  inputData.filter = ''
  onClearMatch()
}

const onClearMatch = () => {
  const changed = !isEmpty(inputData.match)
  inputData.match = ''
  if (changed) {
    emit('matchChanged', inputData.match, inputData.filter, inputData.exact)
  } else {
    emit('filterChanged', inputData.filter, inputData.exact)
  }
}

defineExpose({
  reset: onClearFilter,
})
</script>

<template>
  <n-input-group style="overflow: hidden">
    <slot name="prepend" />
    <n-input
      v-model:value="inputData.filter"
      :placeholder="$t('common.filter.label')"
      :size="props.small ? 'small' : ''"
      :theme-overrides="{ paddingSmall: '0 3px', paddingMedium: '0 6px' }"
      clearable
      @clear="onClearFilter"
      @input="onInput"
      @keyup.enter="onForceFullSearch">
      <template #prefix>
        <slot name="prefix" />
        <n-tooltip v-if="hasMatch" placement="bottom">
          <template #trigger>
            <n-tag closable size="small" @close="onClearMatch" @dblclick="onUpdateMatch">
              {{ inputData.match }}
            </n-tag>
          </template>
          {{
            $t('common.filter.fullSearchResult', {
              pattern: props.useGlob ? inputData.match : '*' + inputData.match + '*',
            })
          }}
        </n-tooltip>
      </template>
      <template #suffix>
        <template v-if="props.useGlob">
          <n-tooltip placement="bottom" trigger="hover">
            <template #trigger>
              <n-tag
                v-model:checked="inputData.exact"
                :checkable="true"
                :type="props.exact ? 'primary' : 'default'"
                size="small"
                strong
                style="padding: 0 5px"
                @updateChecked="onExactChecked">
                <n-icon :size="14">
                  <EqualsIcon />
                </n-icon>
              </n-tag>
            </template>
            <div class="text-block" style="max-width: 600px">
              {{ $t('common.filter.exactMatchTip') }}
            </div>
          </n-tooltip>
        </template>
      </template>
    </n-input>

    <icon-button
      v-if="props.fullSearchIcon"
      :disabled="hasMatch && !hasFilter"
      :icon="props.fullSearchIcon"
      :size="props.small ? 16 : 20"
      :tooltip-delay="1"
      border
      small
      stroke-width="4"
      @click="onFullSearch">
      <template #tooltip>
        <div class="text-block" style="max-width: 600px">
          {{ $t('common.filter.filterPatternTip') }}
        </div>
      </template>
    </icon-button>
    <n-button v-else :disabled="hasMatch && !hasFilter" :focusable="false" @click="onFullSearch">
      {{ $t('common.filter.fullSearch') }}
    </n-button>
    <slot name="append" />
  </n-input-group>
</template>

<style scoped lang="scss">
:deep(.n-input) {
  width: 100%;
  overflow: hidden;
}

:deep(.n-input__prefix) {
  max-width: 50%;

  & > div {
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

:deep(.n-tag__content) {
  overflow: hidden;
  max-width: 100%;
}
</style>

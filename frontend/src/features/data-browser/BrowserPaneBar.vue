<script lang="ts" setup>
import FilterInput from '@/features/common/FilterInput.vue'
import { reactive, ref } from 'vue'
import { MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

const browserStore = useDataBrowserStore()

const filterInputRef = ref(null)
const loading = ref(false)

const filterForm = reactive({
  type: '',
  exact: false,
  pattern: '',
  filter: '',
})

const onReload = async () => {
  try {
    loading.value = true
  } catch (e) {
    console.warn(e)
  } finally {
    loading.value = false
  }
}

const onFilterInput = (val: string, exact: boolean) => {
  filterForm.filter = val
  filterForm.exact = exact
}

const onMatchInput = (matchVal: string, filterVal: string, exact: boolean) => {
  filterForm.pattern = matchVal
  filterForm.filter = filterVal
  filterForm.exact = exact
  onReload()
}
</script>

<template>
  <div class="flex-box-h nav-pane-func" style="height: 36px">
    <FilterInput
      ref="filterInputRef"
      :debounce-delay-ms="1000"
      :full-search-icon="MagnifyingGlassIcon"
      small
      use-glob
      @filter-changed="onFilterInput"
      @match-changed="onMatchInput" />
  </div>
</template>

<style lang="scss" scoped></style>

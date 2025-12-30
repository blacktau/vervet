<script setup lang="ts">
import { type SelectOption, useThemeVars } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { computed, h, reactive, ref } from 'vue'
import dayjs from 'dayjs'
import { map, size, split, uniqBy } from 'lodash'
import IconButton from '@/components/common/IconButton.vue'
import { ArrowPathIcon, TrashIcon } from '@heroicons/vue/24/outline'

interface CmdHistoryItem {
  timestamp: number
  server: string
  cmd: string
}

interface LogData {
  loading: boolean
  server: string
  keyword: string
  history: CmdHistoryItem[]
}

const themeVars = useThemeVars()
const i18n = useI18n()
const data = reactive<LogData>({
  loading: false,
  server: '',
  keyword: '',
  history: [],
})

const filterServerOptions = computed(() => {
  const serverSet = uniqBy(data.history, 'server')
  const options = map(serverSet, ({ server }: CmdHistoryItem) => ({
    label: server,
    value: server,
  }))
  options.splice(0, 0, {
    label: 'common.all',
    value: '',
  })
  return options
})

const tableRef = ref(null)

const columns = computed(() => [
  {
    title: () => i18n.t('log.exec_time'),
    key: 'timestamp',
    defaultSortOrder: 'ascend',
    sorter: 'default',
    width: 180,
    align: 'center',
    titleAlign: 'center',
    render: ({ timestamp }: CmdHistoryItem, index: number) => {
      return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
    },
  },
  {
    title: () => i18n.t('log.server'),
    key: 'server',
    filterOptionValue: data.server,
    filter: (value: string, row: CmdHistoryItem) => {
      return value === '' || row.server === value.toString()
    },
    width: 150,
    align: 'center',
    titleAlign: 'center',
    ellipsis: {
      tooltip: true,
    },
  },
  {
    title: () => i18n.t('log.cmd'),
    key: 'cmd',
    titleAlign: 'center',
    filterOptionValue: data.keyword,
    resizable: true,
    filter: (value: string, row: CmdHistoryItem) => {
      return value === '' || !!~row.cmd.indexOf(value.toString())
    },
    render: ({ cmd }: CmdHistoryItem, index: number) => {
      const cmdList = split(cmd, '\n')
      if (size(cmdList) > 1) {
        return h(
          'div',
          null,
          map(cmdList, (c) => h('div', { class: 'cmd-line' }, c)),
        )
      }
      return h('div', { class: 'cmd-line' }, cmd)
    },
  },
])
</script>

<template>
  <div class="content-log content-container content-value fill-height flex-box-v">
    <n-h3>{{ $t('log.title') }}</n-h3>
    <n-form :disabled="data.loading" class="flex-item" inline>
      <n-form-item :label="$t('log.filter_server')">
        <n-select
          v-model:value="data.server"
          :consistent-menu-width="false"
          :options="filterServerOptions"
          :render-label="({ label, value }: SelectOption) => (value === '' ? $t(label) : label)"
          style="min-width: 100px" />
      </n-form-item>
      <n-form-item :label="$t('log.filter_keyword')">
        <n-input v-model:value="data.keyword" placeholder="" clearable />
      </n-form-item>
      <n-form-item label="&nbsp;">
        <icon-button :icon="ArrowPathIcon" border t-tooltip="log.refresh" />
      </n-form-item>
      <n-form-item label="&nbsp;">
        <icon-button :icon="TrashIcon" border t-tooltip="log.clear_log" />
      </n-form-item>
    </n-form>
    <n-data-table
      ref="tableRef"
      :columns="columns"
      :data="data.history"
      :loading="data.loading"
      class="flex-item-expand"
      flex-height
      virtual-scroll />
  </div>
</template>

<style scoped lang="scss">
@use '@/css/content';
</style>

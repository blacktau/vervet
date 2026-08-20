<script lang="ts" setup>
import { computed, nextTick, ref, watch, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useThemeVars } from 'naive-ui'
import { CircleStackIcon, EyeIcon } from '@heroicons/vue/24/outline'
import CollectionIcon from '@/features/icon/CollectionIcon.vue'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { DataNodeType } from '@/features/data-browser/types.ts'
import {
  type NamespaceRow,
  searchNamespaces,
} from '@/features/data-browser/namespaceSearch.ts'

const dialogStore = useDialogStore()
const browserStore = useDataBrowserStore()
const tabStore = useTabStore()
const { t } = useI18n()
const themeVars = useThemeVars()

const query = ref('')
const highlighted = ref(0)

// Row elements, so the highlighted one can be scrolled into view when the
// keyboard moves it past the edge of the scrolling list.
const rowRefs = ref<(HTMLElement | null)[]>([])

const setRowRef = (el: Element | ComponentPublicInstance | null, index: number) => {
  rowRefs.value[index] = el as HTMLElement | null
}

watch(highlighted, (index) => {
  nextTick(() => {
    rowRefs.value[index]?.scrollIntoView({ block: 'nearest' })
  })
})

const show = computed({
  get: () => dialogStore.isVisible(DialogType.NamespaceFinder),
  set: (value: boolean) => {
    if (!value) {
      dialogStore.closeNamespaceFinder()
    }
  },
})

const status = computed(() => browserStore.getInventoryStatus(tabStore.currentTabId))

const results = computed(() => searchNamespaces(query.value, browserStore.searchIndex))

watch(results, (rows) => {
  highlighted.value = 0
  // Drop refs to rows that no longer exist, so a shrinking result list cannot
  // leave stale elements behind the highlight index.
  rowRefs.value.length = rows.length
})

const iconFor = (row: NamespaceRow) => {
  if (row.type === DataNodeType.Database) {
    return CircleStackIcon
  }
  if (row.type === DataNodeType.View) {
    return EyeIcon
  }
  return CollectionIcon
}

// Rebuilds the tree key the row corresponds to. Database nodes key on
// server:db; collections and views sit under their folder.
const keyFor = (row: NamespaceRow): string => {
  if (row.type === DataNodeType.Database) {
    return `${row.serverID}:${row.db}`
  }
  const folder = row.type === DataNodeType.View ? 'Views' : 'Collections'
  return `${row.serverID}:${row.db}:${folder}:${row.name}`
}

const accept = async (row: NamespaceRow | undefined) => {
  if (!row) {
    return
  }

  dialogStore.closeNamespaceFinder()

  const key = keyFor(row)
  await browserStore.revealNode(row.serverID, key)
  browserStore.openQueryForKey(key, row.type)
}

const onKeydown = (e: KeyboardEvent) => {
  if (results.value.length === 0) {
    return
  }

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    highlighted.value = (highlighted.value + 1) % results.value.length
    return
  }

  if (e.key === 'ArrowUp') {
    e.preventDefault()
    highlighted.value = (highlighted.value - 1 + results.value.length) % results.value.length
    return
  }

  if (e.key === 'Enter') {
    e.preventDefault()
    accept(results.value[highlighted.value])
  }
}
</script>

<template>
  <n-modal v-model:show="show" class="namespace-finder">
    <n-card :bordered="false" size="small" style="width: 600px">
      <n-input
        v-model:value="query"
        :placeholder="t('dataBrowser.finder.placeholder')"
        autofocus
        @keydown="onKeydown" />

      <div v-if="status === 'building'" class="finder-note">
        {{ t('dataBrowser.finder.indexing') }}
      </div>
      <div v-else-if="status === 'error'" class="finder-note">
        {{ t('dataBrowser.finder.indexError') }}
      </div>

      <div class="finder-results">
        <div v-if="query.length === 0" class="finder-note">
          {{ t('dataBrowser.finder.empty') }}
        </div>
        <div v-else-if="results.length === 0" class="finder-note">
          {{ t('dataBrowser.finder.noResults') }}
        </div>
        <div
          v-for="(row, index) in results"
          :key="keyFor(row)"
          :ref="(el) => setRowRef(el, index)"
          :class="['finder-row', { 'finder-row-active': index === highlighted }]"
          :style="index === highlighted ? { backgroundColor: themeVars.hoverColor } : undefined"
          @click="accept(row)"
          @mouseenter="highlighted = index">
          <n-icon :component="iconFor(row)" size="18" />
          <span class="finder-name">{{ row.name }}</span>
          <span class="finder-db">{{ row.db }}</span>
        </div>
      </div>
    </n-card>
  </n-modal>
</template>

<style scoped lang="scss">
.finder-results {
  margin-top: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.finder-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: 3px;
  cursor: pointer;
}

// The highlight colour is applied inline on the row rather than here: n-modal
// teleports its content to body, so the custom property Vue's v-bind() injects
// on this component's root element never reaches the rendered rows.
.finder-row-active {
  font-weight: 500;
}

.finder-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.finder-db {
  opacity: 0.6;
  font-size: 0.85em;
}

.finder-note {
  padding: 8px;
  opacity: 0.6;
}
</style>

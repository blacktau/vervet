<script lang="ts" setup>
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useThemeVars } from 'naive-ui'
import type { DropdownOption } from 'naive-ui'
import { get } from 'lodash'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { extraTheme } from '@/utils/extraTheme.ts'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { ChevronDownIcon, ServerIcon } from '@heroicons/vue/24/outline'
import type { ServerTabItem } from '@/types/ServerTabItem.ts'
import Color from 'color'

const tabStore = useTabStore()
const serverStore = useServerStore()
const settingsStore = useSettingsStore()
const themeVars = useThemeVars()

const tabsRef = ref<HTMLElement | null>(null)
const isOverflowing = ref(false)
let resizeObserver: ResizeObserver | null = null

function checkOverflow() {
  const el = tabsRef.value?.$el ?? tabsRef.value
  if (!el) {
    return
  }
  const scrollWrapper = el.querySelector('.n-tabs-nav-scroll-content')
  const scrollContainer = el.querySelector('.n-tabs-nav-scroll-wrapper')
  if (scrollWrapper && scrollContainer) {
    isOverflowing.value = scrollWrapper.scrollWidth > scrollContainer.clientWidth
  }
}

onMounted(() => {
  resizeObserver = new ResizeObserver(checkOverflow)
  const el = tabsRef.value?.$el ?? tabsRef.value
  if (el) {
    resizeObserver.observe(el)
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})

watch(() => tabStore.tabs.length, () => nextTick(checkOverflow))

const serverDropdownOptions = computed<DropdownOption[]>(() => {
  return tabStore.tabs.map((tab, index) => ({
    label: tab.title,
    key: index,
  }))
})

function handleServerSelect(index: number) {
  tabStore.setActiveTabIndex(index)
}

const onCloseTab = (tabIndex: number) => {
  const tab = get(tabStore.tabs, tabIndex)
  tabStore.closeTab(tab.serverId)
}

const tabBackgroundColor = computed(() => {
  const { serverId } = tabStore?.currentTab || {}
  if (serverId == null) {
    return ''
  }

  const { colour = '' } = serverStore.findServerById(serverId) || {}
  return colour
})

const tabClass = (idx: number) => {
  if (tabStore.activeTabIndex === idx) {
    return [
      'value-tab',
      'value-tab-active',
      tabBackgroundColor.value ? 'value-tab-active-mark' : '',
    ]
  } else if (tabStore.activeTabIndex - 1 === idx) {
    return ['value-tab', 'value-tab-inactive']
  } else {
    return ['value-tab', 'value-tab-inactive', 'value-tab-inactive2']
  }
}

const calcTabColor = (tab: ServerTabItem) => {
  const { colour = '' } = serverStore.findServerById(tab.serverId) || {}
  if (colour === '') {
    return colour
  }

  if (settingsStore.isDark) {
    return Color(colour).darken(0.8)
  }

  return Color(colour).lighten(0.1)
}

const exThemeVars = computed(() => {
  return extraTheme(settingsStore.isDark)
})
</script>

<template>
  <n-tabs
    ref="tabsRef"
    v-model:value="tabStore.activeTabIndex"
    :closeable="true"
    :tabs-padding="3"
    :theme-overrides="{
      tabGapSmallCard: 0,
      tabGapMediumCard: 0,
      tabGapLargeCard: 0,
      tabTextColorCard: themeVars.closeIconColor,
    }"
    size="small"
    type="card"
    @close="onCloseTab"
    @update:value="(tabIndex: number) => tabStore.setActiveTabIndex(tabIndex)">
    <template v-if="isOverflowing" #suffix>
      <n-dropdown
        :options="serverDropdownOptions"
        trigger="click"
        @select="handleServerSelect">
        <n-button quaternary size="tiny" style="margin-left: 4px; --wails-draggable: none">
          <template #icon>
            <n-icon>
              <ChevronDownIcon />
            </n-icon>
          </template>
        </n-button>
      </n-dropdown>
    </template>
    <n-tab
      v-for="(t, index) in tabStore.tabs"
      :key="index"
      :class="tabClass(index)"
      :closable="true"
      :name="index"
      :style="{ backgroundColor: calcTabColor(t) }"
      @dblclick.stop="() => {}">
      <n-space :size="5" :wrap-item="false" align="center" inline justify="center">
        <n-icon size="18">
          <ServerIcon />
        </n-icon>
        <n-ellipsis style="max-width: 150px">{{ t.title }}</n-ellipsis>
      </n-space>
    </n-tab>
  </n-tabs>
</template>

<style lang="scss" scoped>
.value-tab {
  --wails-draggable: none;
  position: relative;
  border: 1px solic v-bind('exThemeVars.splitColor') !important;
}
.value-tab-active {
  background-color: v-bind('themeVars.tabColor') !important;
  border-bottom-color: v-bind('themeVars.tabColor') !important;

  &_mark {
    border-top: 3px solid v-bind('tabBackgroundColor') !important;
  }
}
.value-tab-inactive {
  border-color: #0000 !important;
  &:hover {
    background-color: v-bind('exThemeVars.splitColor') !important;
  }
}

.value-tab-inactive2 {
  &:after {
    content: '';
    position: absolute;
    top: 25%;
    height: 50%;
    width: 1px;
    background-color: v-bind('themeVars.borderColor');
    right: -2px;
  }
}
</style>

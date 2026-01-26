<script lang="ts" setup>
import { useThemeVars } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { useRender } from '@/utils/render.ts'
import { ref } from 'vue'
import BrowserPaneBar from '@/features/data-browser/BrowserPaneBar.vue'
import DataBrowserTree from '@/features/data-browser/DataBrowserTree.vue'

const themeVars = useThemeVars()
const i18n = useI18n()
const render = useRender()

const tabStore = useTabStore()
const browserStore = useDataBrowserStore()
const serverStore = useServerStore()

const browserTreeRef = ref(null)
const filterInputRef = ref(null)
const loading = ref<boolean>(false)
const fullyLoaded = ref<boolean>(false)
const inCheckedState = ref<boolean>(false)
</script>

<template>
  <div class="nav-pane-container flex-box-v">
    <browser-pane-bar />
    <data-browser-tree :loading="loading" />
  </div>
</template>

<style lang="scss" scoped></style>

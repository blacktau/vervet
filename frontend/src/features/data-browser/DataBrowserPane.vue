<script lang="ts" setup>
import { ref } from 'vue'
import BrowserPaneBar from '@/features/data-browser/BrowserPaneBar.vue'
import DataBrowserTree from '@/features/data-browser/DataBrowserTree.vue'

const loading = ref<boolean>(false)
</script>

<template>
  <div class="nav-pane-container flex-box-v">
    <browser-pane-bar />
    <data-browser-tree :loading="loading" />
  </div>
</template>

<style lang="scss" scoped></style>

<script setup lang="ts">

import RegisteredServers from 'components/left_panel/RegisteredServers.vue';
</script>

<template>

  <q-splitter v-model="splitterModel">
    <template v-slot:before>
      <q-tabs v-model="tab" vertical>
        <q-tab name="servers" icon="storage" />
        <q-tab name="activeConnections" icon="table_rows" />
      </q-tabs>
    </template>
    <template v-slot:after>
      <q-tab-panels
        v-model="tab"
        animated
        vertical
        transition-prev="jump-up"
        transition-next="jump-up"
        >
        <q-tab-panel name="servers">
          <RegisteredServers />
        </q-tab-panel>
        <q-tab-panel name="activeConnections">
          <p>activeConnections</p>
        </q-tab-panel>
      </q-tab-panels>
    </template>
  </q-splitter>

</template>

<style scoped lang="scss">
div.q-splitter {
  height: 100%;
}
</style>

<script setup lang="ts">
import { ref, watch } from 'vue';
import RegisteredServers from 'src/components/servers/RegisteredServers.vue';
import 'assets/logo.svg';
import LeftRibbon from 'src/components/ribbon/LeftRibbon.vue';
import type { RibbonOption } from 'src/components/ribbon/model';
import ConnectionsTree from 'src/components/connections/ConnectionsTree.vue';
import WindowBar from 'src/components/windowbar/WindowBar.vue';

const selectedItem = ref('servers');
const splitterModel = ref(25);

watch(selectedItem, (val) => {
  console.log(val);
});

const ribbonOptions: RibbonOption[] = [
  {
    label: 'Registered Servers',
    value: 'servers',
    icon: 'mdi-server',
  },
  {
    label: 'Open Connections',
    value: 'connections',
    icon: 'mdi-database-outline',
  },
];

function handleRibbonChange(e: string) {
  selectedItem.value = e;
  console.log(e);
}
</script>

<template>
  <q-layout view="hHh lpr lff" class="fullscreen non-selectable">
    <WindowBar />
    <q-drawer show-if-above bordered mini>
      <LeftRibbon :options="ribbonOptions" @change="handleRibbonChange" />
    </q-drawer>
    <q-page-container>
      <q-page class="fit no-wrap">
        <q-splitter
          id="main-splitter"
          v-model="splitterModel"
          before-class="inset-shadow full-height column content-stretch"
        >
          <template v-slot:before>
            <RegisteredServers v-if="selectedItem === 'servers'" />
            <ConnectionsTree v-if="selectedItem === 'connections'" />
          </template>

          <template v-slot:after>
            <div class="q-pa-md">
              <div class="text-h4 q-mb-md">Data Goes Here</div>
            </div>
          </template>
        </q-splitter>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<style lang="scss">
#main-splitter {
  min-height: 100vh;
}
/*
.q-item {
  color: darken($primary, 20%);
}

.q-item:hover {
  text-shadow: 0px 0px 10px rgba($primary, 0.8);
  color: lighten($primary, 10%);
}
 */
</style>

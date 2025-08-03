<script setup lang="ts">
import * as SystemProxy from 'app/wailsjs/go/api/SystemProxy';
import { onMounted, ref, watchEffect } from 'vue';
import { api } from 'app/wailsjs/go/models';
import * as runtime from 'app/wailsjs/runtime/runtime';

const operatingSystem = ref<api.OperatingSystem>(api.OperatingSystem.WINDOWS);
const isMax = ref(false);
const isMin = ref(false);

async function getOS() {
  const r = await SystemProxy.GetOs();
  if (r.isSuccess) {
    return r.data;
  }

  return api.OperatingSystem.WINDOWS;
}

function exit() {
  runtime.Quit();
}

function toggleMinimize() {
  if (isMin.value) {
    runtime.WindowUnminimise();
  } else {
    runtime.WindowMinimise();
  }
}

function toggleMaximize() {
  if (isMax.value) {
    runtime.WindowUnmaximise();
  } else {
    runtime.WindowMaximise();
  }
}

onMounted(async () => {
  operatingSystem.value = await getOS();
  isMax.value = await runtime.WindowIsMaximised();
  isMin.value = await runtime.WindowIsMinimised();
});

watchEffect(() => {
  console.log('isMax', isMax.value);
});
</script>

<template>
  <q-header elevated>
    <q-toolbar>
      <q-btn
        v-if="operatingSystem === api.OperatingSystem.OSX"
        dense
        flat
        round
        icon="lens"
        size="8.5px"
        color="red"
      />
      <q-btn
        v-if="operatingSystem === api.OperatingSystem.OSX"
        dense
        flat
        round
        icon="lens"
        size="8.5px"
        color="yellow"
      />
      <q-btn
        v-if="operatingSystem === api.OperatingSystem.OSX"
        dense
        flat
        round
        icon="lens"
        size="8.5px"
        color="green"
      />
      <q-img src="/src/assets/logo.svg" style="width: 50px" />

      <q-toolbar-title> Vervet </q-toolbar-title>
      <q-space />
      <q-btn
        :v-if="operatingSystem != api.OperatingSystem.OSX"
        dense
        flat
        icon="minimize"
        @click="toggleMinimize"
      />
      <q-btn
        :v-if="operatingSystem != api.OperatingSystem.OSX"
        dense
        flat
        :icon="!isMax ? 'crop_square' : 'mdi-window-restore'"
        @click="toggleMaximize"
      />
      <q-btn
        :v-if="operatingSystem != api.OperatingSystem.OSX"
        dense
        flat
        icon="mdi-window-close"
        @click="exit"
      />
    </q-toolbar>
  </q-header>
</template>

<style scoped lang="scss"></style>

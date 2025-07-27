<script setup lang="ts">
import * as SystemProxy from 'app/wailsjs/go/api/SystemProxy';
import {onMounted, ref} from 'vue';
import {api} from 'app/wailsjs/go/models';

const operatingSystem = ref<api.OperatingSystem>(api.OperatingSystem.WINDOWS);

async function getOS() {
  const r = await SystemProxy.GetOs()
  if (r.isSuccess) {
    return r.data
  }

  return api.OperatingSystem.WINDOWS
}

onMounted(async () => {
  operatingSystem.value = await getOS();
})

</script>

<template>
  <q-header elevated>
    <q-toolbar>
      <q-btn v-if="operatingSystem === api.OperatingSystem.OSX" dense flat round icon="lens" size="8.5px" color="red" />
      <q-btn v-if="operatingSystem === api.OperatingSystem.OSX" dense flat round icon="lens" size="8.5px" color="yellow" />
      <q-btn v-if="operatingSystem === api.OperatingSystem.OSX" dense flat round icon="lens" size="8.5px" color="green" />
      <q-img src='/src/assets/logo.svg' style="width: 50px;" />

      <q-toolbar-title>
        Vervet
      </q-toolbar-title>
      <q-space />
      <q-btn v-if="operatingSystem != api.OperatingSystem.OSX" dense flat icon="minimize" />
      <q-btn v-if="operatingSystem != api.OperatingSystem.OSX" dense flat icon="crop_square" />
      <q-btn v-if="operatingSystem != api.OperatingSystem.OSX" dense flat icon="close" />
    </q-toolbar>
  </q-header>
</template>

<style scoped lang="scss">

</style>

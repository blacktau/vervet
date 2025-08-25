<script setup lang="ts">
import type { RibbonOption } from 'components/ribbon/model';
import * as runtime from 'app/wailsjs/runtime/runtime';

const props = defineProps<{
  options: RibbonOption[];
}>();

const model = defineModel<string>({ required: true });

function showSettings() {
  runtime.WindowReload();
}
</script>

<template>
  <div class="ribbon">
    <div
      v-for="(option, index) in props.options"
      :key="index"
      class="ribbon-item"
      @click="model = option.value"
      :class="{ selected: option.value === model }"
    >
      <div><q-icon :name="option.icon" /></div>
      <q-tooltip anchor="center right" self="center middle" :offset="[40, 0]" :delay="250">{{
        option.label
      }}</q-tooltip>
    </div>
    <q-space />
    <q-btn
      flat
      round
      dense
      icon="mdi-cog-outline"
      class="q-mb-sm"
      text-color="indigo-10"
      @click="showSettings"
    >
      <q-tooltip anchor="center right" self="center middle" :offset="[30, 0]" :delay="250"
        >Settings</q-tooltip
      >
    </q-btn>
    <q-btn flat round dense icon="mdi-github" class="q-mb-md" text-color="indigo-10">
      <q-tooltip anchor="center right" self="center middle" :offset="[30, 0]" :delay="250"
        >Github</q-tooltip
      >
    </q-btn>
  </div>
</template>
<style scoped lang="scss">
@use 'quasar/src/css/variables' as q;
@use 'sass:map';
// $

.ribbon {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  min-height: 100%;
  align-items: center;
  min-width: 100%;
}
.ribbon-item {
  font-size: 2rem;
  cursor: pointer;
  color: $indigo-10;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 100%;
  align-items: center;
  border-left: 3px solid transparent;
}

.ribbon-item:hover {
  color: $indigo-13;
  text-shadow: 0 0 6px $indigo-11;
}

.ribbon-item:hover.selected {
  color: $indigo-13;
  text-shadow: 0 0 0;
}

.ribbon-item.selected {
  color: $indigo-13;
  border-left: 3px solid $indigo-13;
}

.label {
  //  transform-origin: top left;
  //  transform: rotate(-0.25turn);
  text-wrap: nowrap;
  writing-mode: vertical-lr;
  transform: rotate(0.5turn);
}
</style>

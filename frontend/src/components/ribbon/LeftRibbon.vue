<script setup lang="ts">
import {RibbonOption} from 'components/ribbon/model';
import {ref} from 'vue';

const props = defineProps<{
  options: RibbonOption[]
}>()

const selected = ref(props.options[0].value)

</script>

<template>
  <div class="ribbon">
    <div
      v-for="(option, index) in props.options"
      :key="index"
      class="ribbon-item"
      @click="selected = option.value"
      :class="{ selected: option.value === selected }">
      <div><q-icon :name="option.icon"/></div>
      <q-tooltip anchor="center right" self="center middle" :offset="[40, 0]" :delay="250">{{ option.label }}</q-tooltip>
<!--      <div v-if="selected === option.value" class="label text-h6">{{option.label}}</div>-->
    </div>

  </div>
</template>
<style scoped lang="scss">
@use "quasar/src/css/variables" as q;
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
.ribbon-item  {
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

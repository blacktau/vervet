<script setup lang="ts">
import * as runtime from 'app/wailsjs/runtime/runtime';
const props = defineProps<{
  target: string | undefined;
}>();

const model = defineModel<boolean | null>({ default: null })

function reloadUi() {
  runtime.WindowReload();
}
</script>
<template>
  <q-menu context-menu touch-position no-parent-event :target="props.target" v-model="model" anchor="bottom right">
    <q-list dense>
      <q-item dense clickable v-close-popup @click.left="reloadUi()" class="q-pa-none">
        <q-item-section avatar rounded style="min-width: 36px">
          <q-icon name="mdi-reload" size="xs" />
        </q-item-section>
        <q-item-section>Reload UI</q-item-section>
      </q-item>
    </q-list>
  </q-menu>
</template>

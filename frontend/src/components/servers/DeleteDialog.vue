<script setup lang="ts">
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import { useQuasar } from 'quasar';
import type { RegisteredServerNode } from 'components/servers/models';

const props = defineProps<{
  target: RegisteredServerNode | undefined;
}>();
const emit = defineEmits(['server-node-deleted']);
const model = defineModel({ default: false });
const $q = useQuasar();

const deleteNode = async () => {
  if (!props.target) return;

  try {
    const result = await serversProxy.RemoveNode(props.target.id, props.target.isGroup);
    if (result.isSuccess) {
      $q.notify({ type: 'positive', message: 'Deleted successfully' });
      emit('server-node-deleted');
    } else {
      $q.notify({ type: 'negative', message: `Failed to delete: ${result.error}` });
    }
  } catch (error) {
    const err = error as Error;
    $q.notify({ type: 'negative', message: `Error deleting node: ${err.message}` });
    console.error('Error deleting node:', error);
  }
};
</script>

<template>
  <!-- Confirm Delete Dialog -->
  <q-dialog persistent v-model="model" transition-hide="scale" transition-show="scale">
    <q-card>
      <q-card-section class="row items-center">
        <q-avatar icon="warning" color="warning" text-color="white" />
        <span class="q-ml-sm">
          Are you sure you want to delete "{{ target?.name }}"?
          <span v-if="target?.isGroup"> This will only delete the folder if it's empty. </span>
        </span>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn label="Cancel" color="secondary" v-close-popup />
        <q-btn label="Delete" color="negative" @click="deleteNode" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<style scoped lang="scss"></style>

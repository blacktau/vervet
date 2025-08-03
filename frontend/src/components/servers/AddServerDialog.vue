<script setup lang="ts">
import { ref } from 'vue';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import { useQuasar } from 'quasar';

const emit = defineEmits(['new-server-added']);
const props = defineProps<{
  parentId?: number;
}>();

const $q = useQuasar();

const newConnection = ref({ name: '', uri: '' });

const saveNewConnection = async () => {
  if (
    !newConnection.value.name ||
    newConnection.value.name.trim().length == 0 ||
    !newConnection.value.uri ||
    newConnection.value.uri.trim().length == 0
  ) {
    $q.notify({
      type: 'warning',
      message: 'Connection name and URI are required.',
    });
    return;
  }
  try {
    const result = await serversProxy.SaveServer(
      newConnection.value.name,
      props.parentId ?? 0,
      newConnection.value.uri
    );

    if (result.isSuccess) {
      $q.notify({
        type: 'positive',
        message: 'Connection successfully saved',
      });
      emit('new-server-added');
    } else {
      $q.notify({
        type: 'negative',
        message: `Failed to save connection: ${result.error}`,
      });
    }
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `Error saving connection: ${err.message}`,
    });
    console.error('Error saving connection:', error);
  }
};
</script>

<template>
  <!-- Add Connection Dialog -->
  <q-dialog persistent>
    <q-card style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">Add New Connection</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input
          dense
          v-model="newConnection.name"
          label="Connection Name"
          autofocus
          @keyup.enter="saveNewConnection"
        />
        <q-input
          dense
          v-model="newConnection.uri"
          label="MongoDB Connection URI"
          class="q-mt-sm"
        />
        <div class="text-caption text-grey-7 q-mt-sm">
          Example: `mongodb://user:pass@host:port/db?authSource=admin`
        </div>
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="Cancel" v-close-popup />
        <q-btn flat label="Add Connection" @click="saveNewConnection" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

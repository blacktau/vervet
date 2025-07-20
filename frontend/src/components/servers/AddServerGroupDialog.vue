<script setup lang="ts">
import {ref} from 'vue';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import { useQuasar } from 'quasar'

const props = defineProps<{
  parentId: number | undefined
}>()
const emit = defineEmits(['server-group-added'])

const $q = useQuasar()
const newFolder = ref({ name: '', parentId: props.parentId });

const saveNewFolder = async () => {
  if (!newFolder.value.name) {
    $q.notify({ type: 'warning', message: 'Folder name is required.' });
    return;
  }
  try {
    const result = await serversProxy.CreateGroup(
      newFolder.value.name,
      newFolder.value.parentId
    );
    if (result.isSuccess) {
      $q.notify({ type: 'positive', message: 'Folder created' });
      emit('server-group-added');
    } else {
      $q.notify({ type: 'negative', message: `Failed to create folder: ${result.error}` });
    }
  } catch (error) {
    const err = error as Error
    $q.notify({ type: 'negative', message: `Error creating folder: ${err.message}` });
    console.error('Error creating folder:', error);
  }
};

</script>

<template>
  <!-- Add Folder Dialog -->
  <q-dialog persistent>
    <q-card style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">Create New Folder</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input dense v-model="newFolder.name" label="Folder Name" autofocus @keyup.enter="saveNewFolder" />
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="Cancel" v-close-popup />
        <q-btn flat label="Create Folder" @click="saveNewFolder" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

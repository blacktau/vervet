<script setup lang="ts">
import { onMounted, ref } from 'vue';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import { useQuasar } from 'quasar';
import type { RegisteredServerNode } from './models';

const emit = defineEmits(['new-server-added', 'server-updated']);

const props = defineProps<{
  target: RegisteredServerNode | undefined;
  isEdit: boolean;
}>();

const model = defineModel<boolean>({ default: false });

const $q = useQuasar();

const newConnection = ref({ name: props.isEdit ? props.target?.name : '', uri: '' });

const validateForm = () => {
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
    return false;
  }

  return true;
};

const saveNewConnection = async () => {
  if (!validateForm()) {
    return;
  }

  try {
    const result = await serversProxy.SaveServer(
      props.target?.id ?? 0,
      newConnection.value.name!,
      newConnection.value.uri,
    );

    if (result.isSuccess) {
      $q.notify({
        type: 'positive',
        message: 'Server successfully saved',
      });
      emit('new-server-added');
    } else {
      $q.notify({
        type: 'negative',
        message: `Failed to save server: ${result.error}`,
      });
    }
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `Error saving server: ${err.message}`,
    });
    console.error('Error saving server:', error);
  }
};

const updateConnection = async () => {
  if (!validateForm()) {
    return;
  }

  try {
    const result = await serversProxy.UpdateServer(
      props.target!.id,
      props.target!.parentId,
      newConnection.value.name!,
      newConnection.value.uri,
    );
    if (result.isSuccess) {
      $q.notify({
        type: 'positive',
        message: 'Server successfully updated',
      });
      emit('server-updated');
    } else {
      $q.notify({
        type: 'negative',
        message: `Failed to update server: ${result.error}`,
      });
    }
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `Error updating server: ${err.message}`,
    });
    console.error('Error updating server:', error);
  }
};

onMounted(async () => {
  if (props.isEdit && props.target) {
    try {
      const result = await serversProxy.GetURI(props.target.id);
      if (result.isSuccess) {
        newConnection.value.uri = result.data;
      } else {
        $q.notify({
          type: 'negative',
          message: `Error retrieving connection string for server: ${result.error}`,
        });
      }
    } catch (error: unknown) {
      const err = error as Error;
      $q.notify({
        type: 'negative',
        message: `Error fetching URI for registered server: ${err.message}`,
      });
      console.error('Error fetching URI:', error);
    }
  }
});
</script>

<template>
  <!-- Add Connection Dialog -->
  <q-dialog persistent v-model="model">
    <q-card style="min-width: 350px">
      <q-card-section>
        <div v-if="!isEdit" class="text-h6">Add New Connection</div>
        <div v-else class="text-h6">Edit Connection</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input
          dense
          v-model="newConnection.name"
          label="Connection Name"
          autofocus
          @keyup.enter="saveNewConnection"
        />
        <q-input dense v-model="newConnection.uri" label="MongoDB Connection URI" class="q-mt-sm" />
        <div class="text-caption text-grey-7 q-mt-sm">
          Example: `mongodb://user:pass@host:port/db?authSource=admin`
        </div>
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn label="Cancel" v-close-popup color="secondary" />
        <q-btn
          v-if="!props.isEdit"
          label="Add Connection"
          @click="saveNewConnection"
          color="primary"
        />
        <q-btn v-else label="Update Connection" @click="updateConnection" color="primary" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

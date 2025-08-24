<script setup lang="ts">
import { ref, watch } from 'vue';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import * as connectionsProxy from 'app/wailsjs/go/api/ConnectionsProxy';
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
const testing = ref(false);
const testable = ref(false);
const isValid = ref(false);

const validateForm = () => {
  const validURI = validateURI(newConnection.value.uri);
  testable.value = validURI;

  isValid.value =
    !!newConnection.value.name &&
    newConnection.value.name.trim().length > 0 &&
    !!newConnection.value.uri &&
    newConnection.value.uri.trim().length > 0 &&
    validURI;
  console.log(newConnection.value);
  console.log('validateURI:', validURI);
  console.log('isValid', isValid.value);

  return isValid.value;
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

const validateURI = (val?: string) => {
  if (!val || val.trim().length == 0) {
    return false;
  }

  const trimmed = val.trim();

  if (!(trimmed.startsWith('mongodb://') || trimmed.startsWith('mongodb+srv://'))) {
    return false;
  }

  return true;
};

const testConnection = async () => {
  testing.value = true;
  try {
    const result = await connectionsProxy.TestConnection(newConnection.value.uri);
    if (result.isSuccess) {
      $q.notify({
        type: 'positive',
        message: 'Connection Successful',
      });
    } else {
      $q.notify({
        type: 'negative',
        message: `Failed to connect to '${newConnection.value.uri}': ${result.error}`,
      });
    }
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: err.message,
    });
  }
  testing.value = false;
};

watch(model, async () => {
  if (!model.value) {
    return;
  }

  if (props.isEdit && props.target) {
    try {
      const result = await serversProxy.GetURI(props.target.id);
      if (result.isSuccess) {
        newConnection.value.uri = result.data;
        validateForm();
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
          :rules="[
            (val) => !!val || '* Required',
            (val) => val.trim().length > 2 || 'Minimum length 2',
          ]"
          @keyup.enter="saveNewConnection"
          @blur="validateForm"
        />
        <q-input
          dense
          v-model="newConnection.uri"
          label="MongoDB Connection URI"
          class="q-mt-sm"
          @blur="validateForm"
          :rules="[
            (val: string) => !!val || '* Required',
            (val: string) => validateURI(val) || 'Invalid URI',
          ]"
        />
        <div class="text-caption text-grey-7 q-mt-sm">
          Example: `mongodb://user:pass@host:port/db?authSource=admin`
        </div>
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn
          label="Test"
          color="positive"
          @click="testConnection"
          :loading="testing"
          :disable="!testable"
        />
        <q-btn label="Cancel" v-close-popup color="secondary" :disable="testing" />
        <q-btn
          v-if="!props.isEdit"
          label="Add Connection"
          @click="saveNewConnection"
          color="primary"
          :disable="testing || !isValid"
        />
        <q-btn
          v-else
          label="Update Connection"
          @click="updateConnection"
          color="primary"
          :disable="testing || !isValid"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

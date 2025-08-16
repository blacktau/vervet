<script setup lang="ts">
import { onMounted, ref } from 'vue';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import { useQuasar } from 'quasar';
import type { RegisteredServerNode } from './models';

const props = defineProps<{
  target: RegisteredServerNode | undefined;
  isEdit: boolean;
}>();
const emit = defineEmits(['server-group-added', 'server-group-updated']);
const model = defineModel({ default: false });

const $q = useQuasar();
const groupName = ref(props.isEdit ? props.target!.name : '');

const validateForm = () => {
  if (!groupName.value || groupName.value.trim().length == 0) {
    $q.notify({ type: 'warning', message: 'Group name is required.' });
    return false;
  }

  return true;
};

const saveNewGroup = async () => {
  if (!validateForm()) {
    return;
  }
  try {
    const result = await serversProxy.CreateGroup(groupName.value, props.target?.id ?? 0);
    if (result.isSuccess) {
      $q.notify({ type: 'positive', message: 'Group created' });
      emit('server-group-added');
    } else {
      $q.notify({
        type: 'negative',
        message: `Failed to create group: ${result.error}`,
      });
    }
  } catch (error) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `Error creating group: ${err.message}`,
    });
    console.error('Error creating group:', error);
  }
};

const updateGroup = async () => {
  if (!validateForm()) {
    return;
  }

  try {
    const result = await serversProxy.UpdateGroup(
      props.target!.id,
      props.target!.parentId,
      groupName.value,
    );
    if (result.isSuccess) {
      $q.notify({
        type: 'positive',
        message: 'Group updated',
      });
      emit('server-group-updated');
    } else {
      $q.notify({
        type: 'negative',
        message: `Failed to update group: ${result.error}`,
      });
    }
  } catch (error) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `Error updating group: ${err.message}`,
    });
  }
};

onMounted(() => {
  console.log('ServerGroupDialog target:', props.target, ' isEdit:', props.isEdit);
});
</script>

<template>
  <q-dialog persistent v-model="model">
    <q-card style="min-width: 350px">
      <q-card-section>
        <div v-if="!isEdit" class="text-h6">Create New Group</div>
        <div v-else class="text-h6">Edit Group</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input
          dense
          v-model="groupName"
          label="Group Name"
          autofocus
          @keyup.enter="saveNewGroup"
        />
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn label="Cancel" color="secondary" v-close-popup />
        <q-btn v-if="!isEdit" color="primary" label="Create Group" @click="saveNewGroup" />
        <q-btn v-else color="primary" label="Save Group" @click="updateGroup" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

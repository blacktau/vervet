<script setup lang="ts">
import { ref } from 'vue';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import { useQuasar } from 'quasar';

const props = defineProps<{
  parentId?: number;
}>();
const emit = defineEmits(['server-group-added']);

const $q = useQuasar();
const newGroupName = ref('');

const saveNewGroup = async () => {
  if (!newGroupName.value || newGroupName.value.trim().length == 0) {
    $q.notify({ type: 'warning', message: 'Group name is required.' });
    return;
  }
  try {
    const result = await serversProxy.CreateGroup(
      newGroupName.value,
      props.parentId ?? 0
    );
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
</script>

<template>
  <!-- Add Group Dialog -->
  <q-dialog persistent>
    <q-card style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">Create New Group</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input
          dense
          v-model="newGroupName"
          label="Group Name"
          autofocus
          @keyup.enter="saveNewGroup"
        />
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="Cancel" v-close-popup />
        <q-btn flat label="Create Group" @click="saveNewGroup" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

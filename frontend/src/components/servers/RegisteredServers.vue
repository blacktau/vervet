<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useQuasar } from 'quasar';
import ServerDialog from './ServerDialog.vue';
import type { RegisteredServerNode } from 'app/src/components/servers/models';
import ServerGroupDialog from 'src/components/servers/ServerGroupDialog.vue';
import DeleteDialog from 'src/components/servers/DeleteDialog.vue';
import ServerTree from 'src/components/servers/ServerTree.vue';
import { fetchConnectionNodes } from './api';
import * as connectionsProxy from 'app/wailsjs/go/api/ConnectionsProxy';
import { showError } from 'src/utils/notifications';

const $q = useQuasar();

const selectedNode = ref<RegisteredServerNode>();
const serverDialogVisible = ref(false);
const groupDialogVisible = ref(false);
const confirmDeleteDialog = ref(false);
const isEdit = ref(false);
const connecting = ref(false);
const nodeToDelete = ref<RegisteredServerNode>();
const nodes = ref<RegisteredServerNode[]>([]);

const fetchNodes = async () => {
  const fetchedNodes = await fetchConnectionNodes($q);
  if (fetchedNodes) {
    nodes.value = fetchedNodes;
  }
};

// --- Dialog and Form Handlers ---
const showServerDialog = (editing: boolean, node?: RegisteredServerNode) => {
  if (node) {
    selectedNode.value = node;
  }

  isEdit.value = editing;

  serverDialogVisible.value = true;
};

const onServerAdded = async () => {
  serverDialogVisible.value = false;
  await fetchNodes(); // Refresh tree
};

const onServerUpdated = async () => {
  serverDialogVisible.value = false;
  await fetchNodes(); // Refresh tree
};

const showGroupDialog = (editing: boolean, node?: RegisteredServerNode) => {
  if (node) {
    selectedNode.value = node;
  }

  groupDialogVisible.value = true;
  isEdit.value = editing;
  console.log('showGroupDialog isEdit:', isEdit.value, ' node:', selectedNode.value);
};

const onGroupAdded = async () => {
  groupDialogVisible.value = false;
  await fetchNodes(); // Refresh tree
};

const onGroupUpdated = async () => {
  groupDialogVisible.value = false;
  await fetchNodes(); // Refresh tree
};

const confirmDeleteNode = (node?: RegisteredServerNode) => {
  if (!node) {
    return;
  }
  nodeToDelete.value = node;
  confirmDeleteDialog.value = true;
};

const onServerNodeDeleted = async () => {
  confirmDeleteDialog.value = false;
  await fetchNodes();
};

const editNode = (node?: RegisteredServerNode) => {
  if (!node) return;
  if (node.isGroup) {
    showGroupDialog(true, node);
  } else {
    showServerDialog(true, node);
  }
};

// --- MongoDB Connection Logic ---
const connectToMongo = async (node?: RegisteredServerNode) => {
  console.log('connecting To Mongo', node);
  if (!node) {
    return;
  }

  connecting.value = true;

  const notif = $q.notify({
    type: 'ongoing',
    message: `Connecting to '${node.name}'...`,
  });

  try {
    const result = await connectionsProxy.Connect(node.id);
    console.log('connected:', node.id, 'result:', result);
    if (!result.isSuccess) {
      showError(`Failed to connect to '${node.name}':\n ${result.error}`, notif);
      connecting.value = false;
      return;
    }

    notif({
      type: 'positive',
      message: `Connected to '${node.name}'.`,
    });
  } catch (error) {
    const err = error as Error;
    showError(`Error connecting to server: '${node.name}': ${err.message}`, notif);
    console.error(`Error connecting to server '${node.name}':`, error);
  }

  connecting.value = false;
};

onMounted(async () => {
  await fetchNodes();
});
</script>

<template>
  <q-layout view="hHh lpR fFf" container class="window-height fit">
    <q-header reveal bordered class="bg-primary text-white">
      <q-bar>
        <div class="text-subtitle1">Registered Servers</div>
        <q-space />
        <q-btn flat dense round icon="add" @click="showServerDialog(false)" class="q-me-sm">
          <q-tooltip>Add Server</q-tooltip>
        </q-btn>
        <q-btn flat dense round icon="o_create_new_folder" @click="showGroupDialog(false)">
          <q-tooltip>Add Server Grouping</q-tooltip>
        </q-btn>
      </q-bar>
    </q-header>
    <q-page-container id="rg-container" class="inset-shadow-down column fit">
      <q-page>
        <ServerTree
          :nodes="nodes"
          @delete-node-requested="confirmDeleteNode"
          @connect="connectToMongo"
          @add-server="(node) => showServerDialog(false, node)"
          @add-group="(node) => showGroupDialog(false, node)"
          @edit-node="editNode"
          :disable="connecting"
        />
      </q-page>
    </q-page-container>
  </q-layout>

  <ServerDialog
    @new-server-added="onServerAdded"
    @server-updated="onServerUpdated"
    :target="selectedNode"
    :isEdit="isEdit"
    v-model="serverDialogVisible"
  />
  <ServerGroupDialog
    @server-group-added="onGroupAdded"
    @server-group-updated="onGroupUpdated"
    :target="selectedNode"
    :isEdit="isEdit"
    v-model="groupDialogVisible"
  />
  <DeleteDialog
    @server-node-deleted="onServerNodeDeleted"
    :target="nodeToDelete"
    v-model="confirmDeleteDialog"
  />
</template>

<style scoped lang="scss"></style>

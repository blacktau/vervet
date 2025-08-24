<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useQuasar } from 'quasar';
import ServerDialog from './ServerDialog.vue';
import type { RegisteredServerNode } from 'app/src/components/servers/models';
import ServerGroupDialog from 'src/components/servers/ServerGroupDialog.vue';
import DeleteDialog from 'src/components/servers/DeleteDialog.vue';
import ServerTree from 'src/components/servers/ServerTree.vue';
import { fetchConnectionNodes } from './api';

const $q = useQuasar();

const selectedNode = ref<RegisteredServerNode>();
const serverDialogVisible = ref(false);
const groupDialogVisible = ref(false);
const confirmDeleteDialog = ref(false);
const isEdit = ref(false);
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
const connectToMongo = (node?: RegisteredServerNode) => {
  if (!node) {
    return;
  }

  $q.loading.show({ message: `Connecting to ${node.name}... ${node.id}` });
  // try {
  //   const [success, message] = await connectionManager.Connect(id);
  //   if (success) {
  //     $q.notify({type: 'positive', message: message});
  //     // Add to connected IDs if not already there
  //     if (!connectedClientIDs.value.includes(id)) {
  //       connectedClientIDs.value.push(id);
  //     }
  //     // Open a new tab for this connection if not already open
  //     if (!openConnectionTabs.value.some(tab => tab.id === id)) {
  //       openConnectionTabs.value.push({id: id, name: name, queryResult: null});
  //     }
  //     currentTab.value = `conn-${id}`; // Switch to the new tab
  //   } else {
  //     $q.notify({type: 'negative', message: `Connection failed: ${message}`});
  //   }
  // } catch (error) {
  //   $q.notify({type: 'negative', message: `Error connecting: ${error.message}`});
  //   console.error('Error connecting to MongoDB:', error);
  // } finally {
  //   $q.loading.hide();
  // }
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

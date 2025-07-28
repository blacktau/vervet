<script setup lang="ts">
import { onMounted, ref, watchEffect } from 'vue';
import { useQuasar } from 'quasar';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import AddServerDialog from './AddServerDialog.vue';
import { RegisteredServerNode } from 'components/servers/models';
import AddServerGroupDialog from 'components/servers/AddServerGroupDialog.vue';
import DeleteDialog from 'components/servers/DeleteDialog.vue';
import ServerTree from 'components/servers/ServerTree.vue';

const $q = useQuasar();

const selectedNodeId = ref<number | null>(); // For QTree selection
const addServerDialog = ref(false);
const addGroupDialog = ref(false);
const confirmDeleteDialog = ref(false);
const nodeToDelete = ref<RegisteredServerNode | undefined>();
const nodes = ref<RegisteredServerNode[] | undefined>([]);

// --- Data Fetching and Tree Building ---
const fetchConnectionNodes = async () => {
  try {
    const result = await serversProxy.GetServers();
    if (!result.isSuccess) {
      $q.notify({
        type: 'negative',
        message: `Failed to load Registered Servers: ${result.error}`,
      });
      console.error('Error fetching Registered Servers:', result.error);
      return;
    }
    nodes.value = result.data as RegisteredServerNode[];
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `An error occurred when loading the Registered Servers: ${err.message}`,
    });
    console.error('Error fetching Registered Server nodes:', error);
  }
};

// --- Dialog and Form Handlers ---
const showAddServerDialog = (node?: RegisteredServerNode) => {
  console.log('showAddServerDialog');
  selectedNodeId.value = node?.id || null
  addServerDialog.value = true;
};

const onServerAdded = async () => {
  console.log('onServerAdded');
  addServerDialog.value = false;
  await fetchConnectionNodes(); // Refresh tree
};

const showAddGroupDialog = (node?: RegisteredServerNode) => {
  selectedNodeId.value = node?.id || null
  addGroupDialog.value = true;
};

const onGroupAdded = async () => {
  addGroupDialog.value = false;
  await fetchConnectionNodes(); // Refresh tree
};

const confirmDeleteNode = (node: RegisteredServerNode) => {
  nodeToDelete.value = node;
  confirmDeleteDialog.value = true;
};

const onServerNodeDeleted = async () => {
  confirmDeleteDialog.value = true;
  await fetchConnectionNodes();
};

watchEffect(() => {
  console.log(selectedNodeId.value)
})

// --- MongoDB Connection Logic ---
const connectToMongo = async (node: RegisteredServerNode) => {
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
  await fetchConnectionNodes();
});
</script>

<template>
  <q-layout view="hHh lpR fFf" container class="window-height fit">
    <q-header reveal bordered class="bg-primary text-white">
      <q-bar>
        <div class="text-subtitle1">Registered Servers</div>
        <q-space />
        <q-btn flat dense round icon="add" @click="showAddServerDialog()" class="q-me-sm">
          <q-tooltip>Add Server</q-tooltip>
        </q-btn>
        <q-btn flat dense round icon="create_new_folder" @click="showAddGroupDialog()">
          <q-tooltip>Add Server Grouping</q-tooltip>
        </q-btn>
      </q-bar>
    </q-header>
    <q-page-container id="rg-container" class="inset-shadow-down column fit">
      <q-page>
        <ServerTree :nodes="nodes" @delete-node-requested="confirmDeleteNode" @connect="connectToMongo"
          @add-server="showAddServerDialog" @add-group="showAddGroupDialog" />
      </q-page>
    </q-page-container>
  </q-layout>

  <AddServerDialog @new-server-added="onServerAdded" :parentId="selectedNodeId" v-model="addServerDialog" />
  <AddServerGroupDialog @server-group-added="onGroupAdded" :parentId="selectedNodeId" v-model="addGroupDialog" />
  <DeleteDialog @server-node-deleted="onServerNodeDeleted" :target="nodeToDelete" v-model="confirmDeleteDialog" />
</template>

<style scoped lang="scss"></style>

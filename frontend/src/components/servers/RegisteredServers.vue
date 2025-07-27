<script setup lang="ts">
import {onMounted, ref} from 'vue';
import {useQuasar} from 'quasar';
import * as serversProxy from 'app/wailsjs/go/api/ServersProxy';
import AddServerDialog from './AddServerDialog.vue';
import {RegisteredServerNode} from 'components/servers/models';
import AddServerGroupDialog from 'components/servers/AddServerGroupDialog.vue';
import DeleteDialog from 'components/servers/DeleteDialog.vue';
import ServerTree from 'components/servers/ServerTree.vue';

const $q = useQuasar();

const selectedNode = ref<RegisteredServerNode | null>(null); // For QTree selection
const addServerDialog = ref(false);
const addGroupDialog = ref(false);
const confirmDeleteDialog = ref(false);
const nodeToDelete = ref<RegisteredServerNode | null>(null);
const nodes: RegisteredServerNode[] = ref([]);

// --- Data Fetching and Tree Building ---
const fetchConnectionNodes = async () => {
  try {
    const result = await serversProxy.GetServers();
    if (!result.isSuccess) {
      $q.notify({
        type: 'negative',
        message: `Failed to load Registered Servers: ${result.error}`
      });
      console.error('Error fetching Registered Servers:', result.error);
      return;
    }
    nodes.value = result.data;
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `An error occurred when loading the Registerd Servers: ${err.message}`,
    });
    console.error('Error fetching Registered Server nodes:', error);
  }
};

// --- Dialog and Form Handlers ---
const showAddServerDialog = () => {
  console.log('showAddServerDialog');
  addServerDialog.value = true;
};

const onServerAdded = async () => {
  console.log('onServerAdded');
  addServerDialog.value = false;
  await fetchConnectionNodes(); // Refresh tree
};

const showAddGroupDialog = () => {
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

// --- MongoDB Connection Logic ---
const connectToMongo = async (id: number, name: string) => {

  $q.loading.show({message: `Connecting to ${name}... ${id}`});
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
        <q-space/>
        <q-btn flat dense round icon="add" @click="showAddServerDialog()"
               class="q-me-sm">
          <q-tooltip>Add Server</q-tooltip>
        </q-btn>
        <q-btn flat dense round icon="create_new_folder" @click="showAddGroupDialog()">
          <q-tooltip>Add Server Grouping</q-tooltip>
        </q-btn>
      </q-bar>

    </q-header>
    <q-page-container id="rg-container" class="inset-shadow-down column fit" >
      <q-page>
        <ServerTree :nodes="nodes" @delete-node-requested="confirmDeleteNode" />
      </q-page>
    </q-page-container>
  </q-layout>

  <AddServerDialog @new-server-added="onServerAdded" :parentId="selectedNode" v-model="addServerDialog"/>
  <AddServerGroupDialog @server-group-added="onGroupAdded" :parentId="selectedNode" v-model="addGroupDialog"/>
  <DeleteDialog
    @server-node-deleted="onServerNodeDeleted"

    :target="nodeToDelete" v-model="confirmDeleteDialog"/>

</template>

<style scoped lang="scss">
</style>

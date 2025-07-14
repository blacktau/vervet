<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useQuasar } from 'quasar';
import { configuration } from 'app/wailsjs/go/models';
import { RegisteredServerNode } from 'components/left_panel/models';
import * as registeredServerManager from 'app/wailsjs/go/servers/RegisteredServerManager'

const $q = useQuasar();

const connectionTree = ref<RegisteredServerNode[]>([]);
const selectedNodeId = ref<number>(0); // For QTree selection
const addConnectionDialog = ref(false);
const addFolderDialog = ref(false);
const newConnection = ref({ name: '', uri: '', parentId: 0 });
const newFolder = ref({ name: '', parentId: 0 });
const confirmDeleteDialog = ref(false);
const nodeToDelete = ref<RegisteredServerNode | null>(null);

// --- Data Fetching and Tree Building ---
const fetchConnectionNodes = async () => {
  try {
    const result = await registeredServerManager.GetRegisteredServers()
    if (!result.isSuccess) {
      $q.notify({
        type: 'negative',
        message: `Failed to load Registered Servers: ${result.error}`
      })
      console.error('Error fetching Registered Servers:', result.error)
      return
    }
    const nodes: configuration.RegisteredServer[] = result.data
    connectionTree.value = buildTree(nodes);
  } catch (error: unknown) {
    const err = error as Error
    $q.notify({
      type: 'negative',
      message: `An error occurred when loading the Registerd Servers: ${err.message}`,
    });
    console.error('Error fetching Registered Server nodes:', error);
  }
};


// Helper to build a nested tree structure from a flat list
const buildTree = (nodes: configuration.RegisteredServer[]) => {
  const nodeMap: Record<string, RegisteredServerNode> = {};
  const tree: RegisteredServerNode[] = [];

  nodes.forEach(node => {
    nodeMap[node.id] = { ...node, children: [] };
  });

  nodes.forEach(node => {
    if (node.parentId === 0) {
      tree.push(nodeMap[node.id]);
    } else {
      if (nodeMap[node.parentId]) {
        nodeMap[node.parentId].children.push(nodeMap[node.id]);
        // Sort children: folders first, then connections, then by name
        nodeMap[node.parentId].children.sort((a, b) => {
          if (a.isFolder && !b.isFolder) return -1;
          if (!a.isFolder && b.isFolder) return 1;
          return a.name.localeCompare(b.name);
        });
      }
    }
  });
  // Sort root level: folders first, then connections, then by name
  tree.sort((a, b) => {
    if (a.isFolder && !b.isFolder) return -1;
    if (!a.isFolder && b.isFolder) return 1;
    return a.name.localeCompare(b.name);
  });
  return tree;
};

// --- Dialog and Form Handlers ---
const showAddConnectionDialog = (parentId: number) => {
  newConnection.value = { name: '', uri: '', parentId: parentId };
  addConnectionDialog.value = true;
};

const saveNewConnection = async () => {
  if (!newConnection.value.name || !newConnection.value.uri) {
    $q.notify({ type: 'warning', message: 'Connection name and URI are required.' });
    return;
  }
  try {
    const result = await registeredServerManager.SaveRegisterServer(
      newConnection.value.name,
      newConnection.value.parentId,
      newConnection.value.uri
    );
    if (typeof result === 'boolean' && result) {
      $q.notify({ type: 'positive', message: 'Registered Server successfully saved' });
      addConnectionDialog.value = false;
      await fetchConnectionNodes(); // Refresh tree
    } else if (typeof result === 'string') {
      const message = result
      $q.notify({ type: 'negative', message: `Failed to save connection: ${message}` });
    }
  } catch (error: unknown) {
    const err = error as Error
    $q.notify({ type: 'negative', message: `Error saving connection: ${err.message}` });
    console.error('Error saving connection:', error);
  }
};

const showAddFolderDialog = (parentId: number) => {
  newFolder.value = { name: '', parentId: parentId };
  addFolderDialog.value = true;
};

const saveNewFolder = async () => {
  if (!newFolder.value.name) {
    $q.notify({ type: 'warning', message: 'Folder name is required.' });
    return;
  }
  try {
    const result = await registeredServerManager.CreateFolder(
      newFolder.value.name,
      newFolder.value.parentId
    );
    if (result.isSuccess) {
      $q.notify({ type: 'positive', message: 'Folder created' });
      addFolderDialog.value = false;
      await fetchConnectionNodes(); // Refresh tree
    } else {
      $q.notify({ type: 'negative', message: `Failed to create folder: ${result.error}` });
    }
  } catch (error) {
    const err = error as Error
    $q.notify({ type: 'negative', message: `Error creating folder: ${err.message}` });
    console.error('Error creating folder:', error);
  }
};

const confirmDeleteNode = (node: RegisteredServerNode) => {
  nodeToDelete.value = node;
  confirmDeleteDialog.value = true;
};

const deleteNode = async () => {
  if (!nodeToDelete.value) return;

  try {
    const result = await registeredServerManager.RemoveNode(
      nodeToDelete.value.id,
      nodeToDelete.value.isFolder
    );
    if (result.isSuccess) {
      $q.notify({ type: 'positive', message: 'Deleted successfully' });
      await fetchConnectionNodes(); // Refresh tree
    } else {
      $q.notify({ type: 'negative', message: `Failed to delete: ${result.error}` });
    }
  } catch (error) {
    const err = error as Error
    $q.notify({ type: 'negative', message: `Error deleting node: ${err.message}` });
    console.error('Error deleting node:', error);
  } finally {
    confirmDeleteDialog.value = false;
    nodeToDelete.value = null;
  }
};

// --- MongoDB Connection Logic ---
const connectToMongo = async (id: number, name: string) => {

  $q.loading.show({ message: `Connecting to ${name}... ${id}` });
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
  await fetchConnectionNodes()
})
</script>

<template>
  <q-layout view="hHh lpR fFf">
    <q-header reveal bordered class="bg-primary text-white">
      <q-toolbar>
        <q-toolbar-title>Registered Servers</q-toolbar-title>
      </q-toolbar>
    </q-header>
    <q-page-container>

      <q-tree :nodes="connectionTree" node-key="id" label-key="name" selected-color="primary"
        v-model:selected="selectedNodeId" default-expand-all no-nodes-label="No connections or folders yet. Add one!">

        <template v-slot:default-header="prop">
          <div class="row items-center">
            <q-icon :name="prop.node.isFolder ? 'folder' : 'storage'" color="grey-8" size="20px" class="q-mr-sm" />
            <div class="text-weight-bold">{{ prop.node.name }}</div>
            <q-space />
            <q-btn v-if="!prop.node.isFolder" flat round dense icon="link" color="green" size="sm" class="q-ml-sm">
              <q-tooltip>Connect</q-tooltip>
            </q-btn>
            <q-btn v-if="isConnected(prop.node.id)" flat round dense icon="link_off" color="red" size="sm"
              class="q-ml-sm" @click.stop="disconnectFromMongo(prop.node.id)">
              <q-tooltip>Disconnect</q-tooltip>
            </q-btn>

            <q-btn v-if="prop.node.isFolder" flat round dense icon="add" color="blue" size="sm" class="q-ml-sm"
              @click.stop="showAddConnectionDialog(prop.node.id)">
              <q-tooltip>Add Connection</q-tooltip>
            </q-btn>
            <q-btn v-if="prop.node.isFolder" flat round dense icon="create_new_folder" color="blue" size="sm"
              class="q-ml-sm" @click.stop="showAddFolderDialog(prop.node.id)">
              <q-tooltip>Add Subfolder</q-tooltip>
            </q-btn>
            <q-btn flat round dense icon="delete" color="negative" size="sm" class="q-ml-sm"
              @click.stop="confirmDeleteNode(prop.node)">
              <q-tooltip>Delete</q-tooltip>
            </q-btn>
          </div>
        </template>
      </q-tree>
    </q-page-container>
  </q-layout>

  <!-- Add Connection Dialog -->
  <q-dialog v-model="addConnectionDialog" persistent>
    <q-card style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">Add New Connection</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input dense v-model="newConnection.name" label="Connection Name" autofocus
          @keyup.enter="saveNewConnection" />
        <q-input dense v-model="newConnection.uri" label="MongoDB Connection URI" class="q-mt-sm" />
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

  <!-- Add Folder Dialog -->
  <q-dialog v-model="addFolderDialog" persistent>
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

  <!-- Confirm Delete Dialog -->
  <q-dialog v-model="confirmDeleteDialog" persistent>
    <q-card>
      <q-card-section class="row items-center">
        <q-avatar icon="warning" color="warning" text-color="white" />
        <span class="q-ml-sm">
          Are you sure you want to delete "{{ nodeToDelete.name }}"?
          <span v-if="nodeToDelete.isFolder">
            This will only delete the folder if it's empty.
          </span>
        </span>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="Cancel" color="primary" v-close-popup />
        <q-btn flat label="Delete" color="negative" @click="deleteNode" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<style scoped lang="scss"></style>

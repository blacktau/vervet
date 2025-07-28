<script setup lang="ts">
import { ref, watch, watchEffect } from 'vue';
import { RegisteredServerNode } from 'components/servers/models';
import { configuration } from 'app/wailsjs/go/models';

const props = defineProps<{
  nodes: RegisteredServerNode[];
}>();

const emit = defineEmits<{
  (e: 'delete-node-requested', node: RegisteredServerNode | null): void;
  (e: 'connect', node: RegisteredServerNode | null): void;
  (e: 'add-server', node: RegisteredServerNode | null): void;
  (e: 'add-group', node: RegisteredServerNode | null): void;
}>();

const selectedNode = ref<RegisteredServerNode | null>(null);
const connectionTree = ref<RegisteredServerNode[]>([]);
const showMenu = ref(false);
const menuTarget = ref<string | undefined>()

// Helper to build a nested tree structure from a flat list
const buildTree = (nodes: configuration.RegisteredServer[]) => {
  const nodeMap: Record<string, RegisteredServerNode> = {};
  const tree: RegisteredServerNode[] = [];

  if (!nodes) {
    return tree
  }

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
          if (a.isGroup && !b.isGroup) return -1;
          if (!a.isGroup && b.isGroup) return 1;
          return a.name.localeCompare(b.name);
        });
      }
    }
  });
  // Sort root level: folders first, then connections, then by name
  tree.sort((a, b) => {
    if (a.isGroup && !b.isGroup) return -1;
    if (!a.isGroup && b.isGroup) return 1;
    return a.name.localeCompare(b.name);
  });
  return tree;
};

watchEffect(() => {
  console.log('servers fetched', props.nodes);
  connectionTree.value = buildTree(props.nodes);
});

function confirmDeleteNode() {
  if (!selectedNode.value) return;
  emit('delete-node-requested', selectedNode.value);
}

function addServer() {
  if (!selectedNode.value) return;
  emit('add-server', selectedNode.value)
}

function addGroup() {
  if (!selectedNode.value) return;
  emit('add-group', selectedNode.value)
}

function connect(node: RegisteredServerNode) {
  if (node && !node.isGroup) {
    emit('connect', node)
  }
}

function rightClickTarget(node: RegisteredServerNode, e: PointerEvent) {
  selectedNode.value = node
  const div = e.currentTarget as Element
  menuTarget.value = '#' + div.id
  showMenu.value = true
  e.preventDefault()
}

</script>

<template>
  <div>
    <q-tree :nodes="connectionTree" node-key="id" label-key="name" selected-color="primary" default-expand-all
      v-model:selected="selectedNode" no-nodes-label="No connections or folders yet. Add one!" class="fit no-wrap"
      @contextmenu="(e: PointerEvent) => e.preventDefault()">

      <template v-slot:default-header="prop">
        <div class="row items-center no-wrap fit" @contextmenu="(e: PointerEvent) => rightClickTarget(prop.node, e)"
          :id="'node_' + prop.node.id">
          <q-icon v-if="prop.node.isGroup" name="folder" color="orange" size="20px" class="q-mr-sm" />
          <q-icon v-else name="mdi-database-outline" color="green" size="20px" class="q-mr-sm" />
          <div class="text-weight-bold">{{ prop.node.name }}</div>
          <!-- <q-space /> -->
          <!-- <q-btn v-if="!prop.node.isGroup" flat round dense icon="link" color="green" size="sm" class="q-ml-sm"> -->
          <!--   <q-tooltip>Connect</q-tooltip> -->
          <!-- </q-btn> -->
          <!---->
          <!-- <q-btn v-if="prop.node.isGroup" flat round dense icon="add" color="blue" size="sm" class="q-ml-sm" -->
          <!--   @click.stop="showAddServerDialog(prop.node.id)"> -->
          <!--   <q-tooltip>Add Connection</q-tooltip> -->
          <!-- </q-btn> -->
          <!-- <q-btn v-if="prop.node.isGroup" flat round dense icon="create_new_folder" color="blue" size="sm" class="q-ml-sm" -->
          <!--   @click.stop="showAddGroupDialog(prop.node.id)"> -->
          <!--   <q-tooltip>Add Subfolder</q-tooltip> -->
          <!-- </q-btn> -->
          <!-- <q-btn flat round dense icon="delete" color="negative" size="sm" class="q-ml-sm" -->
          <!--   @click.stop="confirmDeleteNode(prop.node)"> -->
          <!--   <q-tooltip>Delete</q-tooltip> -->
          <!-- </q-btn> -->
        </div>
      </template>
    </q-tree>
    <q-menu context-menu touch-postition v-model="showMenu" :target="menuTarget" no-parent-event>
      <q-list dense style="min-width: 100px">
        <q-item clickable v-close-popup @click="connect(selectedNode)" v-if="!selectedNode.isGroup">
          <q-item-section>Connect</q-item-section>
        </q-item>
        <q-item clickable v-close-popup @click="addServer(selectedNode)" v-if="selectedNode.isGroup">
          <q-item-section>Add Server</q-item-section>
        </q-item>
        <q-item clickable v-close-popup @click="addGroup(selectedNode)" v-if="selectedNode.isGroup">
          <q-item-section>Add Group</q-item-section>
        </q-item>
        <q-item clickable v-close-popup @click="confirmDeleteNode(selectedNode)"
          :disable="selectedNode.isGroup && selectedNode.children.length > 0">
          <q-item-section>Delete</q-item-section>
        </q-item>
      </q-list>

    </q-menu>
  </div>
</template>

<style scoped lang="scss"></style>

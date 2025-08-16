<script setup lang="ts">
import { ref, watchEffect } from 'vue';
import type { RegisteredServerNode } from 'src/components/servers/models';
import type { configuration } from 'app/wailsjs/go/models';
import ServerTreeContextMenu from './ServerTreeContextMenu.vue';

const props = defineProps<{
  nodes: RegisteredServerNode[];
}>();

const emit = defineEmits<{
  (e: 'delete-node-requested', node: RegisteredServerNode | undefined): void;
  (e: 'connect', node: RegisteredServerNode | undefined): void;
  (e: 'add-server', node: RegisteredServerNode | undefined): void;
  (e: 'add-group', node: RegisteredServerNode | undefined): void;
  (e: 'edit-node', node: RegisteredServerNode | undefined): void;
}>();

const selectedNode = ref<RegisteredServerNode>();
const connectionTree = ref<RegisteredServerNode[]>([]);
const showMenu = ref(false);
const menuTarget = ref<string | undefined>();

// Helper to build a nested tree structure from a flat list
function buildTree(nodes: configuration.RegisteredServer[]) {
  const nodeMap: Record<string, RegisteredServerNode> = {};
  const tree: RegisteredServerNode[] = [];

  if (!nodes) {
    return tree;
  }

  nodes.forEach((node) => {
    nodeMap[node.id] = {
      ...node,
      children: [],
      header: node.isGroup ? 'group' : 'connection',
      showButtons: false,
    };
  });

  nodes.forEach((node) => {
    if (node.parentId === 0) {
      const tNode = nodeMap[node.id];
      if (tNode) {
        tree.push(tNode);
      }
    } else {
      const parent = nodeMap[node.parentId];
      if (parent) {
        const child = nodeMap[node.id];

        if (child) {
          parent.children.push(child);
        }

        // Sort children: folders first, then connections, then by name
        parent.children.sort((a, b) => {
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
}

watchEffect(() => {
  connectionTree.value = buildTree(props.nodes);
});

function confirmDeleteNode(node?: RegisteredServerNode) {
  if (!node) return;
  emit('delete-node-requested', node);
}

function addServer(node?: RegisteredServerNode) {
  if (!node) return;
  emit('add-server', node);
}

function addGroup(node?: RegisteredServerNode) {
  if (!node) return;
  emit('add-group', node);
}

function connect(node?: RegisteredServerNode) {
  if (node && !node.isGroup) {
    emit('connect', node);
  }
}

function editNode(node?: RegisteredServerNode) {
  if (!node) return;
  emit('edit-node', node);
}

function showContextMenu(node: RegisteredServerNode) {
  selectedNode.value = node;
  menuTarget.value = '#node_' + node.id;
  showMenu.value = true;
}

function showButtons(node: RegisteredServerNode) {
  return node.showButtons || node.id == selectedNode.value?.id;
}

function selectNode(node: RegisteredServerNode) {
  if (selectedNode.value?.id === node.id) {
    selectedNode.value = undefined;
    return;
  }

  selectedNode.value = node;
}
</script>

<template>
  <div>
    <q-tree
      :nodes="connectionTree"
      node-key="id"
      label-key="name"
      :duration="100"
      :selected="selectedNode?.id"
      no-nodes-label="No connections or folders yet. Add one!"
      class="fit no-wrap"
      @contextmenu.prevent="() => {}"
    >
      <template v-slot:header-group="prop">
        <div
          :id="'node_' + prop.node.id"
          class="row items-center no-wrap fit cursor-pointer tree-node q-mr-sm"
          @mouseenter="prop.node.showButtons = true"
          @mouseleave="prop.node.showButtons = false"
          @click.left="selectNode(prop.node)"
          @contextmenu.prevent="showContextMenu(prop.node)"
        >
          <q-icon
            :name="prop.expanded ? 'mdi-folder-open-outline' : 'mdi-folder-outline'"
            color="orange"
            size="20px"
            class="q-mr-sm"
          />
          <div class="text-weight-bold">{{ prop.node.name }}</div>
          <q-space />
          <q-btn
            v-if="showButtons(prop.node)"
            flat
            round
            color="secondary"
            size="xs"
            icon="mdi-cog-outline"
            @click.prevent="editNode(prop.node)"
          />
          <q-btn
            v-if="showButtons(prop.node)"
            flat
            round
            color="negative"
            size="xs"
            icon="mdi-trash-can-outline"
            :disable="prop.node.children.length != 0"
            @click.prevent="confirmDeleteNode(prop.node)"
          />
        </div>
      </template>
      <template v-slot:header-connection="prop">
        <div
          class="row items-center no-wrap fit cursor-pointer tree-node q-pr-sm"
          :id="'node_' + prop.node.id"
          @mouseenter="prop.node.showButtons = true"
          @mouseleave="prop.node.showButtons = false"
          @click.left="selectNode(prop.node)"
          @contextmenu.prevent="showContextMenu(prop.node)"
        >
          <q-icon name="mdi-database-outline" color="green" size="20px" class="q-mr-sm" />
          <div class="text-weight-bold">{{ prop.node.name }}</div>
          <q-space />
          <q-btn
            v-if="showButtons(prop.node)"
            flat
            round
            color="positive"
            size="xs"
            icon="mdi-connection"
            @click.prevent="connect(prop.node)"
          />
          <q-btn
            v-if="showButtons(prop.node)"
            flat
            round
            color="secondary"
            size="xs"
            icon="mdi-cog-outline"
            @click="editNode(prop.node)"
          />
          <q-btn
            v-if="showButtons(prop.node)"
            flat
            round
            color="negative"
            size="xs"
            icon="mdi-trash-can-outline"
            @click.prevent="confirmDeleteNode(prop.node)"
          />
        </div>
      </template>
    </q-tree>
    <ServerTreeContextMenu
      v-model="showMenu"
      :target-node="selectedNode"
      :target-selector="menuTarget"
      @add-server="addServer"
      @add-group="addGroup"
      @edit-node="editNode"
      @delete-node="confirmDeleteNode"
      @connect="connect"
    />
  </div>
</template>

<style scoped lang="scss">
@use 'sass:color';
:deep(.q-tree__node--selected) {
  background: $blue-1;
  .q-tree__node-header-content {
    color: $primary;
  }
}
.tree-node {
  min-height: 24px;
}
</style>

<script setup lang="ts">
import { ref, watchEffect } from 'vue';
import type { RegisteredServerNode } from 'src/components/servers/models';
import type { configuration } from 'app/wailsjs/go/models';

const props = defineProps<{
  nodes: RegisteredServerNode[];
}>();

const emit = defineEmits<{
  (e: 'delete-node-requested', node: RegisteredServerNode | undefined): void;
  (e: 'connect', node: RegisteredServerNode | undefined): void;
  (e: 'add-server', node: RegisteredServerNode | undefined): void;
  (e: 'add-group', node: RegisteredServerNode | undefined): void;
}>();

const selectedNode = ref<RegisteredServerNode>();
const connectionTree = ref<RegisteredServerNode[]>([]);
const showMenu = ref(false);
const menuTarget = ref<string | undefined>();

// Helper to build a nested tree structure from a flat list
const buildTree = (nodes: configuration.RegisteredServer[]) => {
  const nodeMap: Record<string, RegisteredServerNode> = {};
  const tree: RegisteredServerNode[] = [];

  if (!nodes) {
    return tree;
  }

  nodes.forEach((node) => {
    nodeMap[node.id] = { ...node, children: [] };
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
};

watchEffect(() => {
  console.log('servers fetched', props.nodes);
  connectionTree.value = buildTree(props.nodes);
});

function confirmDeleteNode(node?: RegisteredServerNode) {
  if (!node) return;
  emit('delete-node-requested', selectedNode.value);
}

function addServer(node?: RegisteredServerNode) {
  if (!node) return;
  emit('add-server', selectedNode.value);
}

function addGroup(node?: RegisteredServerNode) {
  if (!node) return;
  emit('add-group', selectedNode.value);
}

function connect(node?: RegisteredServerNode) {
  if (node && !node.isGroup) {
    emit('connect', node);
  }
}

function editNode(node?: RegisteredServerNode) {
  if (!node) return;
  console.log('edit-node');
}

function rightClickTarget(node: RegisteredServerNode, e: MouseEvent) {
  selectedNode.value = node;
  const div = e.currentTarget as Element;
  menuTarget.value = '#' + div.id;
  showMenu.value = true;
  e.preventDefault();
}
</script>

<template>
  <div>
    <q-tree
      :nodes="connectionTree"
      node-key="id"
      label-key="name"
      selected-color="primary"
      default-expand-all
      v-model:selected="selectedNode"
      no-nodes-label="No connections or folders yet. Add one!"
      class="fit no-wrap"
      @contextmenu="(e: MouseEvent) => e.preventDefault()"
    >
      <template v-slot:default-header="prop">
        <div
          class="row items-center no-wrap fit"
          @contextmenu="(e: MouseEvent) => rightClickTarget(prop.node, e)"
          :id="'node_' + prop.node.id"
        >
          <q-icon
            v-if="prop.node.isGroup"
            :name="prop.expanded ? 'mdi-folder-open-outline' : 'mdi-folder-outline'"
            color="orange"
            size="20px"
            class="q-mr-sm"
          />
          <q-icon v-else name="mdi-database-outline" color="green" size="20px" class="q-mr-sm" />
          <div class="text-weight-bold">{{ prop.node.name }}</div>
        </div>
      </template>
    </q-tree>
    <q-menu context-menu touch-postition v-model="showMenu" :target="menuTarget" no-parent-event>
      <q-list dense style="min-width: 100px">
        <q-item
          clickable
          v-close-popup
          @click="connect(selectedNode)"
          v-if="!selectedNode?.isGroup"
        >
          <q-item-section>Connect</q-item-section>
        </q-item>
        <q-item
          clickable
          v-close-popup
          @click="addServer(selectedNode)"
          v-if="selectedNode?.isGroup"
        >
          <q-item-section avatar><q-icon name="add" /></q-item-section>
          <q-item-section>Add Connection</q-item-section>
        </q-item>
        <q-item
          clickable
          v-close-popup
          @click="addGroup(selectedNode)"
          v-if="selectedNode?.isGroup"
        >
          <q-item-section avatar><q-icon name="create-new-folder-outline" /></q-item-section>
          <q-item-section>Add Group</q-item-section>
        </q-item>
        <q-item clickable v-close-popup @click="editNode(selectedNode)">
          <q-item-section>Edit</q-item-section>
        </q-item>
        <q-item
          clickable
          v-close-popup
          @click="confirmDeleteNode(selectedNode)"
          :disable="selectedNode?.isGroup && selectedNode.children.length > 0"
        >
          <q-item-section>Delete</q-item-section>
        </q-item>
      </q-list>
    </q-menu>
  </div>
</template>

<style scoped lang="scss"></style>

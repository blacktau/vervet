<script setup lang="ts">
import {ref, watch, watchEffect} from 'vue';
import {RegisteredServerNode} from 'components/servers/models';
import {configuration} from 'app/wailsjs/go/models';

const props = defineProps<{
  nodes: RegisteredServerNode[];
}>()

const emit = defineEmits<{
  (e: 'node-selected', node: RegisteredServerNode | null): void;
  (e: 'delete-node-requested', node: RegisteredServerNode | null): void;
}>()

const selectedNode = ref<RegisteredServerNode | null>(null);
const connectionTree = ref<RegisteredServerNode[]>([]);

watch(selectedNode, () => {
  console.log('tree node selected', selectedNode.value);
  emit('node-selected', selectedNode.value);
})

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
  console.log('servers fetched', props.nodes)
  connectionTree.value = buildTree(props.nodes)
})

function confirmDeleteNode(node: RegisteredServerNode) {
  emit('delete-node-requested', node)
}

</script>

<template>
  <q-tree :nodes="connectionTree" node-key="id" label-key="name" selected-color="primary"
          v-model:selected="selectedNode" default-expand-all no-nodes-label="No connections or folders yet. Add one!"
  class="fit no-wrap">

    <template v-slot:default-header="prop">
      <div class="row items-center no-wrap fit">
        <q-icon :name="prop.node.isGroup ? 'folder' : 'storage'" color="grey-8" size="20px" class="q-mr-sm" />
        <div class="text-weight-bold">{{ prop.node.name }}</div>
        <q-space />
        <q-btn v-if="!prop.node.isGroup" flat round dense icon="link" color="green" size="sm" class="q-ml-sm">
          <q-tooltip>Connect</q-tooltip>
        </q-btn>

        <q-btn v-if="prop.node.isGroup" flat round dense icon="add" color="blue" size="sm" class="q-ml-sm"
               @click.stop="showAddServerDialog(prop.node.id)">
          <q-tooltip>Add Connection</q-tooltip>
        </q-btn>
        <q-btn v-if="prop.node.isGroup" flat round dense icon="create_new_folder" color="blue" size="sm"
               class="q-ml-sm" @click.stop="showAddGroupDialog(prop.node.id)">
          <q-tooltip>Add Subfolder</q-tooltip>
        </q-btn>
        <q-btn flat round dense icon="delete" color="negative" size="sm" class="q-ml-sm"
               @click.stop="confirmDeleteNode(prop.node)">
          <q-tooltip>Delete</q-tooltip>
        </q-btn>
      </div>
    </template>
  </q-tree>
</template>

<style scoped lang="scss">

</style>

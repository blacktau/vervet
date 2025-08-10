<script setup lang="ts">
import type { RegisteredServerNode } from './models';

const props = defineProps<{
  targetSelector: string | undefined;
  targetNode: RegisteredServerNode | undefined;
}>();

const model = defineModel<boolean | null>({ default: null });

const emit = defineEmits<{
  (e: 'delete-node', node: RegisteredServerNode | undefined): void;
  (e: 'connect', node: RegisteredServerNode | undefined): void;
  (e: 'add-server', node: RegisteredServerNode | undefined): void;
  (e: 'add-group', node: RegisteredServerNode | undefined): void;
  (e: 'edit-node', node: RegisteredServerNode | undefined): void;
}>();

function connect() {
  emit('connect', props.targetNode);
}

function addServer() {
  emit('add-server', props.targetNode);
}

function addGroup() {
  emit('add-group', props.targetNode);
}

function deleteNode() {
  emit('delete-node', props.targetNode);
}

function editNode() {
  emit('edit-node', props.targetNode);
}

function isGroup() {
  return props.targetNode?.isGroup;
}

function hasChildren() {
  if (!props.targetNode) return false;
  return props.targetNode.children.length > 0;
}
</script>

<template>
  <q-menu
    context-menu
    touch-postition
    v-model="model"
    :target="props.targetSelector"
    no-parent-event
  >
    <q-list dense>
      <q-item dense clickable v-close-popup @click="connect" v-if="!isGroup()">
        <q-item-section avatar rounded class="q-pa-xs" style="min-width: 30px"
          ><q-icon name="mdi-connection" size="xs"
        /></q-item-section>
        <q-item-section>Connect</q-item-section>
      </q-item>
      <q-item dense clickable v-close-popup @click="addServer" v-if="isGroup()">
        <q-item-section avatar rounded style="min-width: 30px"
          ><q-icon name="add" size="xs"
        /></q-item-section>
        <q-item-section>Add Connection</q-item-section>
      </q-item>
      <q-item dense clickable v-close-popup @click="addGroup()" v-if="isGroup()">
        <q-item-section avatar rounded style="min-width: 30px"
          ><q-icon name="o_create_new_folder" size="xs"
        /></q-item-section>
        <q-item-section>Add Group</q-item-section>
      </q-item>
      <q-item dense clickable v-close-popup @click="editNode()">
        <q-item-section avatar rounded style="min-width: 30px"
          ><q-icon name="mdi-cog-outline" size="xs"
        /></q-item-section>
        <q-item-section>Edit</q-item-section>
      </q-item>
      <q-item
        dense
        clickable
        v-close-popup
        @click="deleteNode()"
        :disable="isGroup() && hasChildren()"
      >
        <q-item-section avatar rounded style="min-width: 30px"
          ><q-icon name="mdi-trash-can-outline" size="xs"
        /></q-item-section>
        <q-item-section>Delete</q-item-section>
      </q-item>
    </q-list>
  </q-menu>
</template>

<template>
    <q-header elevated>
      <q-toolbar>
        <q-img src='/src/assets/logo.svg' style="width: 50px;"/>

        <q-toolbar-title>
          Vervet
        </q-toolbar-title>

        <q-space />

        <q-btn flat dense round icon="add" label="New Connection" @client="showAddConnectionDialog(0)" class="q-me-sm"/>
        <q-btn flat dense round icon="folder" label="New Folder" @click="showAddFolderDialog(0)" />

      </q-toolbar>
    </q-header>
</template>

<script lang="ts" setup>
import 'assets/logo.svg'

</script>

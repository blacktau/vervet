<template>
  <q-layout view="hHh lpr lff" class="fullscreen">
    <AppHeader />
    <q-drawer v-model="drawer" show-if-above :mini="miniState" @mouseenter="miniState = false"
      @mouseleave="miniState = true" mini-to-overlay :width="250" :breakpoint="250" bordered>
      <q-list padding>
        <q-item clickable v-ripple :active="leftPanel === 'servers'" @click="leftPanel = 'servers'">
          <q-item-section avatar>
            <q-icon name="storage" />
          </q-item-section>
          <q-item-section>
            Registered Servers
          </q-item-section>
        </q-item>
        <q-item clickable v-ripple :active="leftPanel === 'connections'" @click="leftPanel = 'connections'">
          <q-item-section avatar>
            <q-icon name="account_tree" />
          </q-item-section>
          <q-item-section>
            Connections
          </q-item-section>
        </q-item>
      </q-list>
    </q-drawer>
    <q-page-container>
      <q-page padding>
        <q-splitter v-model="splitterModel">
          <template v-slot:before>
            <RegisteredServers />
          </template>

          <template v-slot:after>
            <div class="q-pa-md">
              <div class="text-h4 q-mb-md">Data Goes Here</div>
            </div>
          </template>
        </q-splitter>
      </q-page>
    </q-page-container>
  </q-layout>
</template>

<script>
import { defineComponent, onMounted, ref } from 'vue';
import { useQuasar } from 'quasar';
import AppHeader from 'components/AppHeader.vue';
import RegisteredServers from 'components/left_panel/RegisteredServers.vue';

export default defineComponent({
  name: 'App',
  components: { RegisteredServers, AppHeader },
  // Renamed to App for main layout
  setup() {
    const $q = useQuasar();

    // Example Test Query (will be expanded later)
    // --- Lifecycle Hook ---
    onMounted(async () => {
      // Get initially connected clients (if the app somehow persisted connections or reconnected)
      // This is a good place to call GetConnectedClientIDs if your Go backend
      // maintains a list of connections across app restarts (e.g., re-establishing from previous session)
      // For this example, we assume connections are ephemeral unless explicitly connected.
    });

    return {
      drawer: ref(false),
      miniState: ref(true),
      leftPanel: ref('servers'),
      splitterModel: ref(20),
    };
  },
});
</script>

<style lang="scss">
// Optional: Add some basic styling if needed
.q-tree {
  .q-tree__node-header {
    padding: 8px 16px;
  }
}
</style>

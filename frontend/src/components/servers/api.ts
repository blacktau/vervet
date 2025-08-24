import { GetServers } from 'app/wailsjs/go/api/ServersProxy';
import type { QVueGlobals } from 'quasar';
import type { RegisteredServerNode } from './models';

// --- Data Fetching and Tree Building ---
export const fetchConnectionNodes = async ($q: QVueGlobals) => {
  try {
    const result = await GetServers();
    if (!result.isSuccess) {
      $q.notify({
        type: 'negative',
        message: `Failed to load Registered Servers: ${result.error}`,
      });
      console.error('Error fetching Registered Servers:', result.error);
      return;
    }

    return result.data as RegisteredServerNode[];
  } catch (error: unknown) {
    const err = error as Error;
    $q.notify({
      type: 'negative',
      message: `An error occurred when loading the Registered Servers: ${err.message}`,
    });
    console.error('Error fetching Registered Server nodes:', error);
  }

  return undefined;
};

import { Notify } from 'quasar';
import type { QNotifyUpdateOptions } from 'quasar';

export function showError(msg: string, notif?: (props?: QNotifyUpdateOptions) => void) {
  const creator = !notif ? Notify.create : notif;
  const dismiss = creator({
    type: 'negative',
    icon: 'error',
    message: msg,
    multiLine: true,
    timeout: 0,
    actions: [
      {
        label: 'Dismiss',
        color: 'white',
        handler: () => {
          if (notif) {
            notif();
            return;
          }

          if (dismiss) {
            dismiss();
          }
        },
      },
    ],
  });
}

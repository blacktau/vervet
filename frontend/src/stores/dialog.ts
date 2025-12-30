import { defineStore } from 'pinia'
import { type servers } from 'wailsjs/go/models'

export enum DialogMode {
  New = 0,
  Edit
}

type DialogState = {
  visible: boolean,
  type: DialogMode
  serverDetails?: servers.RegisteredServer,
  data?: unknown
}

export const enum DialogType {
  Server = 'server',
  Group = 'group',
  Settings = 'settings',
  About = 'about'
}

export type ServerDialogData = {
  serverId: string,
  mode: 'edit' | 'clone'
}

export const useDialogStore = defineStore('dialog', {
  state: () => ({
    dialogs: {
      [DialogType.Server]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.Group]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.Settings]: {
        visible: false,
      }
    } as Record<DialogType, DialogState>,
  }),
  actions: {
    openNewDialog(dialog: DialogType, data?: unknown) {
      if (!this.dialogs[dialog]) {
        this.dialogs[dialog] = {
          visible: true,
          type: DialogMode.New,
          data: data,
        }

        return
      }

      this.dialogs[dialog].visible = true
      this.dialogs[dialog].type = DialogMode.New
      this.dialogs[dialog].data = data
    },
    closeDialog(dialog: DialogType) {
      if (!this.dialogs[dialog]) {
        return
      }

      this.dialogs[dialog] = {
        visible: false,
        type: DialogMode.New
      }
    },
    openEditDialog<T>(dialog: DialogType, data: T) {
      if (!this.dialogs[dialog]) {
        this.dialogs[dialog] = {
          visible: true,
          type: DialogMode.Edit,
          data: data,
        }
        return
      }

      this.dialogs[dialog].visible = true
      this.dialogs[dialog].type = DialogMode.Edit
      this.dialogs[dialog].data = data
    },
    openNewServerDialog() {
      this.openNewDialog(DialogType.Server, undefined)
    },
    async openServerEditDialog(serverId: string) {
      this.openEditDialog(DialogType.Server, {
        serverId: serverId,
        mode: 'edit',
      })
    },
    async openCloneServerDialog(serverID: string) {
      this.openEditDialog(DialogType.Server, {
        serverId: serverID,
        mode: 'clone',
      })
    },
    openNewGroupDialog() {
      this.openNewDialog(DialogType.Group, undefined)
    },
    closeNewGroupDialog() {
      this.closeDialog(DialogType.Group)
    },
    openSettingsDialog(tag: string = '') {
      this.openNewDialog(DialogType.Settings, tag)
    },
    closeSettingsDialog() {
      this.closeDialog(DialogType.Settings)
    },
    openAboutDialog() {
      this.openNewDialog(DialogType.About)
    },
    closeAboutDialog() {
      this.closeDialog(DialogType.About)
    },
    openRenameGroupDialog(groupId: string) {
      this.openEditDialog(DialogType.Group, groupId)
    },
    closeRenameGroupDialog() {
      this.closeDialog(DialogType.Group)
    }
  },
  getters: {
    serverDialogData(state): ServerDialogData | undefined {
      const dialog = state.dialogs[DialogType.Server]
      if (!dialog) {
        return undefined
      }

      return dialog.data as ServerDialogData
    }
  },
})

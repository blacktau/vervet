import { defineStore } from 'pinia'
import { type models } from 'wailsjs/go/models.ts'

export enum DialogMode {
  New = 0,
  Edit,
}

type DialogState = {
  visible: boolean
  type: DialogMode
  serverDetails?: models.RegisteredServer
  data?: unknown
}

export const enum DialogType {
  Server = 'server',
  Group = 'group',
  Settings = 'settings',
  About = 'about',
}

export type ServerDialogData = {
  serverId: string
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
        type: DialogMode.New,
      },
      [DialogType.About]: {
        visible: false,
      },
    } as Record<DialogType, DialogState>,
  }),
  actions: {
    showNewDialog(dialog: DialogType, data?: unknown) {
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
    hide(dialog: DialogType) {
      if (!this.dialogs[dialog]) {
        return
      }

      this.dialogs[dialog] = {
        visible: false,
        type: DialogMode.New,
      }
    },
    showEditDialog<T>(dialog: DialogType, data: T) {
      console.log('showEditDialog', dialog, data)
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
    showNewServerDialog() {
      this.showNewDialog(DialogType.Server, undefined)
    },
    async showServerEditDialog(serverId: string) {
      this.showEditDialog(DialogType.Server, {
        serverId: serverId,
        mode: 'edit',
      })
    },
    async showCloneServerDialog(serverID: string) {
      this.showEditDialog(DialogType.Server, {
        serverId: serverID,
        mode: 'clone',
      })
    },
    openNewGroupDialog() {
      this.showNewDialog(DialogType.Group, undefined)
    },
    closeNewGroupDialog() {
      this.hide(DialogType.Group)
    },
    openSettingsDialog(tag: string = '') {
      this.showNewDialog(DialogType.Settings, tag)
    },
    closeSettingsDialog() {
      this.hide(DialogType.Settings)
    },
    openAboutDialog() {
      this.showNewDialog(DialogType.About)
    },
    closeAboutDialog() {
      this.hide(DialogType.About)
    },
    openRenameGroupDialog(groupId: string) {
      this.showEditDialog(DialogType.Group, groupId)
    },
    closeRenameGroupDialog() {
      this.hide(DialogType.Group)
    },
  },
  getters: {
    serverDialogData(state): ServerDialogData | undefined {
      const dialog = state.dialogs[DialogType.Server]
      if (!dialog) {
        return undefined
      }

      return dialog.data as ServerDialogData
    },
    isVisible(state) {
      return (dialog: DialogType) => {
        return state.dialogs[dialog].visible
      }
    },
    getDialogData(state) {
      return <T>(dialog: DialogType): T | undefined => {
        const data = state.dialogs[dialog].data as T | undefined
        return data
      }
    },
  },
})

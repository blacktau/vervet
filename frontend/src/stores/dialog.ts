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
  AddDatabase = 'addDatabase',
  AddCollection = 'addCollection',
  CreateIndex = 'createIndex',
  RenameCollection = 'renameCollection',
  ServerPicker = 'serverPicker',
  Export = 'export',
  ExportResults = 'exportResults',
  DestructiveConfirm = 'destructiveConfirm',
}

export type ServerDialogData = {
  serverId: string
  mode: 'edit' | 'clone'
}

export type DestructiveConfirmKind = 'database' | 'collection'

export type DestructiveConfirmData = {
  kind: DestructiveConfirmKind
  name: string
  impact: {
    collectionCount?: number
    documentCount?: number
  }
  onConfirm: () => Promise<void>
}

export type ExportResultsData = {
  ejson: string
  collectionName?: string
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
      [DialogType.AddDatabase]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.AddCollection]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.CreateIndex]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.RenameCollection]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.ServerPicker]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.Export]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.ExportResults]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
      [DialogType.DestructiveConfirm]: {
        visible: false,
        type: DialogMode.New,
      } as DialogState,
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
    async showCloneServerDialog(serverId: string) {
      this.showEditDialog(DialogType.Server, {
        serverId: serverId,
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
    openAddDatabaseDialog(serverID: string) {
      this.showNewDialog(DialogType.AddDatabase, serverID)
    },
    closeAddDatabaseDialog() {
      this.hide(DialogType.AddDatabase)
    },
    openAddCollectionDialog(serverID: string, dbName: string) {
      this.showNewDialog(DialogType.AddCollection, { serverID, dbName })
    },
    closeAddCollectionDialog() {
      this.hide(DialogType.AddCollection)
    },
    openCreateIndexDialog(serverID: string, dbName: string, collectionName: string) {
      this.showNewDialog(DialogType.CreateIndex, { serverID, dbName, collectionName })
    },
    openEditIndexDialog(serverID: string, dbName: string, collectionName: string, index: unknown) {
      this.showEditDialog(DialogType.CreateIndex, { serverID, dbName, collectionName, index })
    },
    closeCreateIndexDialog() {
      this.hide(DialogType.CreateIndex)
    },
    openRenameCollectionDialog(serverID: string, dbName: string, collectionName: string) {
      this.showNewDialog(DialogType.RenameCollection, { serverID, dbName, collectionName })
    },
    closeRenameCollectionDialog() {
      this.hide(DialogType.RenameCollection)
    },
    openServerPickerDialog(data?: unknown) {
      this.showNewDialog(DialogType.ServerPicker, data)
    },
    openDestructiveConfirmDialog(data: DestructiveConfirmData) {
      this.showNewDialog(DialogType.DestructiveConfirm, data)
    },
    closeDestructiveConfirmDialog() {
      this.hide(DialogType.DestructiveConfirm)
    },
    openExportResultsDialog(data: ExportResultsData) {
      this.showNewDialog(DialogType.ExportResults, data)
    },
    closeExportResultsDialog() {
      this.hide(DialogType.ExportResults)
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
    exportResultsData(state): ExportResultsData {
      const data = state.dialogs[DialogType.ExportResults]?.data as ExportResultsData | undefined
      return data ?? { ejson: '' }
    },
  },
})

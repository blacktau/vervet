import type { TreeOption } from 'naive-ui'

export enum DataNodeType {
  Server,
  Database,
  Folder,
  Collection,
  View,
}

export type DataTreeNode = TreeOption & {
  type: DataNodeType
}

export interface ContextMenuOption {
  label: string
  key: string
  disabled?: boolean
  children?: ContextMenuOption[]
}

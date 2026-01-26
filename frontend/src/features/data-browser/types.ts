import type { TreeOption } from 'naive-ui'

export enum DataNodeType {
  Server,
  Database,
  Folder,
  Collection,
  IndexCollection,
  Index,
}

export type DataTreeNode = TreeOption & {
  type: DataNodeType
}

import type { TreeOption } from 'naive-ui'


export enum ServerNodeType {
  Group = 0,
  Server,
}

export type ServerTreeNode = TreeOption & {
  type: ServerNodeType
  isSrv?: boolean
  isCluster?: boolean
  color?: string
  path: string
}


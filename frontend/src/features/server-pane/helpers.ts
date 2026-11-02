import type { TreeSelectOption } from 'naive-ui'
import type { RegisteredServerNode } from '@/features/server-pane/serverStore.ts'

export const filterGroupMap = (node: RegisteredServerNode) => {
  if (!node.isGroup) {
    return undefined
  }

  if (node.children == null) {
    return node
  }

  const children: RegisteredServerNode[] = []
  for (let i = 0, ln = node.children.length; i < ln; ++i) {
    if (!node.children[i]?.isGroup) {
      continue
    }

    const child = filterGroupMap(node.children[i]!)
    if (child) {
      children.push(child)
    }
  }

  return { ...node, children }
}

export const toTreeSelectOptions = (nodes: RegisteredServerNode[]): TreeSelectOption[] =>
  nodes.map((node) => ({
    ...node,
    children: node.children && toTreeSelectOptions(node.children),
  }))

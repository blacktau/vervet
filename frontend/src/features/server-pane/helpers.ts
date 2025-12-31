import type { RegisteredServerNode } from '@/features/server-pane/serverStore.ts'
import type { TreeSelectOption } from 'naive-ui'

export const filterGroupMap = (node: RegisteredServerNode) => {
  if (!node.isGroup) {
    return undefined
  }

  const children: TreeSelectOption[] = []
  for (let i = 0, ln = node.children.length; i < ln; ++i) {
    if (!node.children[i]?.isGroup) {
      continue
    }

    const child = filterGroupMap(node.children[i]!)
    if (child) {
      children.push(child)
    }
  }

  return {
    label: node.name,
    key: node.id,
    children: children,
  } as TreeSelectOption
}


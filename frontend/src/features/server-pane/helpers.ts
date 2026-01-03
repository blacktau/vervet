import type { RegisteredServerNode } from '@/features/server-pane/serverStore.ts'
import type { TreeSelectOption } from 'naive-ui'
import { hexGammaCorrection, parseHexColor, toHexColor } from '@/utils/colours.ts'
import type { ServerTreeNode } from '@/features/server-pane/types.ts'

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

export const getServerColour = (server: ServerTreeNode) => {
  if (server == null || server.colour == null || server.colour.length == 0) {
    return undefined
  }

  const rgb = parseHexColor(server.colour)
  const darker = hexGammaCorrection(rgb, 0.75)
  console.log('getServerColour', server.colour, darker)
  return toHexColor(darker)
}

import type { RegisteredServerNode } from '@/features/server-pane/serverStore.ts'
import type { TreeSelectOption } from 'naive-ui'
import { hexGammaCorrection, parseHexColor, toHexColor } from '@/utils/colours.ts'
import type { ServerTreeNode } from '@/features/server-pane/types.ts'

export const filterGroupMap = (node: RegisteredServerNode) => {
  if (!node.isGroup) {
    return undefined
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

export const getServerColour = (server: RegisteredServerNode, selected: boolean = false) => {
  if (server == null || server.colour == null || server.colour.length == 0) {
    return undefined
  }

  const rgb = parseHexColor(server.colour)
  let gamma = 0.20
  if (selected) {
    gamma = 0.35
  }
  const darker = hexGammaCorrection(rgb, gamma)
  return toHexColor(darker)
}

import type { RegisteredServerNode } from '@/features/server-pane/serverStore.ts'
import Color from 'color'

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

export const getServerColour = (
  server: RegisteredServerNode,
  selected: boolean = false,
  isDark: boolean,
) => {
  if (server == null || server.colour == null || server.colour.length == 0) {
    return undefined
  }

  if (isDark) {
    let gamma = 0.7
    if (selected) {
      gamma = 0.5
    }
    return Color(server.colour).darken(gamma).hex()
  }

  let gamma = 0.4
  if (selected) {
    gamma = 0.3
  }

  return Color(server.colour).lighten(gamma).hex()
}

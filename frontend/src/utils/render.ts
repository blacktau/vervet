import { h, type VNodeChild } from 'vue'
import { type IconProps, NIcon } from 'naive-ui'

export function useRender() {
  return {
    renderIcon: (icon: string | (() => VNodeChild) | undefined, props: IconProps = {}) => {
      if (icon == null) {
        return undefined
      }
      return h(NIcon, null, { default: () => h(icon, props) })
    },
    renderLabel: (label: string, props = {}) => {
      console.log('renderLabel', label)
      return h('div', props, label)
    },
  }
}

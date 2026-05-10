import { onBeforeUnmount, onMounted, type Ref, type ComponentPublicInstance } from 'vue'
import Sortable from 'sortablejs'

type TabRoot = HTMLElement | ComponentPublicInstance | null

export function useTabSortable(
  rootRef: Ref<TabRoot>,
  navSelector: string,
  onReorder: (from: number, to: number) => void,
): void {
  let sortable: Sortable | null = null

  onMounted(() => {
    const root = (rootRef.value as ComponentPublicInstance | null)?.$el
      ?? (rootRef.value as HTMLElement | null)
    if (!root) {
      return
    }
    const navEl = root.querySelector(navSelector) as HTMLElement | null
    if (!navEl) {
      return
    }
    sortable = new Sortable(navEl, {
      animation: 150,
      scroll: true,
      draggable: '.n-tabs-tab-wrapper',
      filter: '.n-tabs-tab__close, .n-tabs-tab-pad',
      preventOnFilter: false,
      onEnd(evt) {
        const from = evt.oldIndex
        const to = evt.newIndex
        if (typeof from === 'number' && typeof to === 'number' && from !== to) {
          onReorder(from, to)
        }
      },
    })
  })

  onBeforeUnmount(() => {
    sortable?.destroy()
    sortable = null
  })
}

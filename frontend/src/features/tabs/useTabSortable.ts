import { nextTick, onBeforeUnmount, watch, type Ref, type ComponentPublicInstance } from 'vue'
import Sortable from 'sortablejs'

type TabRoot = HTMLElement | ComponentPublicInstance | null

export function useTabSortable(
  rootRef: Ref<TabRoot>,
  navSelector: string,
  onReorder: (from: number, to: number) => void,
): void {
  let sortable: Sortable | null = null
  let boundEl: HTMLElement | null = null

  function teardown() {
    sortable?.destroy()
    sortable = null
    boundEl = null
  }

  function setup() {
    const value = rootRef.value
    const root = (value as ComponentPublicInstance | null)?.$el
      ?? (value as HTMLElement | null)
    if (!root) {
      teardown()
      return
    }
    const navEl = root.querySelector(navSelector) as HTMLElement | null
    if (!navEl) {
      teardown()
      return
    }
    if (sortable && boundEl === navEl) {
      return
    }
    teardown()
    boundEl = navEl
    sortable = new Sortable(navEl, {
      animation: 150,
      scroll: true,
      direction: 'horizontal',
      forceFallback: true,
      fallbackOnBody: false,
      fallbackTolerance: 5,
      fallbackClass: 'tab-sortable-fallback',
      ghostClass: 'tab-sortable-ghost',
      revertOnSpill: true,
      draggable: '.n-tabs-tab-wrapper',
      filter: '.n-tabs-tab__close, .n-tabs-tab-pad',
      preventOnFilter: false,
      onEnd(evt) {
        const from = evt.oldDraggableIndex
        const to = evt.newDraggableIndex
        if (typeof from === 'number' && typeof to === 'number' && from !== to) {
          onReorder(from, to)
        }
      },
    })
  }

  watch(rootRef, () => {
    void nextTick(setup)
  }, { immediate: true, flush: 'post' })

  onBeforeUnmount(teardown)
}

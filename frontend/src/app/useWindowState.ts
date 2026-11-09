import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Events, Window } from '@wailsio/runtime'

const changeEvents = [
  Events.Types.Common.WindowMaximise,
  Events.Types.Common.WindowUnMaximise,
  Events.Types.Common.WindowFullscreen,
  Events.Types.Common.WindowUnFullscreen,
  Events.Types.Common.WindowRestore,
  Events.Types.Common.WindowDidResize,
]

/** Tracks whether the main window is maximised or fullscreen. */
export function useWindowState() {
  const maximised = ref(false)
  const fullscreen = ref(false)
  const stops: (() => void)[] = []

  const refresh = async () => {
    try {
      const [isFullscreen, isMaximised] = await Promise.all([Window.IsFullscreen(), Window.IsMaximised()])
      fullscreen.value = isFullscreen
      maximised.value = isMaximised
    } catch {
      // Outside the app window (e.g. a plain browser in dev) there is no state to read.
    }
  }

  onMounted(() => {
    for (const name of changeEvents) {
      stops.push(Events.On(name, () => void refresh()))
    }
    void refresh()
  })

  onBeforeUnmount(() => {
    for (const stop of stops) {
      stop()
    }
  })

  return { maximised, fullscreen }
}

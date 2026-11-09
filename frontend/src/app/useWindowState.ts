import { onBeforeUnmount, onMounted, ref } from 'vue'
import { Events, Window } from '@wailsio/runtime'

// Wails reports every maximise and fullscreen change through these, however
// it was made (buttons, keyboard or the window manager), so resizes aren't needed.
const changeEvents = [
  Events.Types.Common.WindowMaximise,
  Events.Types.Common.WindowUnMaximise,
  Events.Types.Common.WindowFullscreen,
  Events.Types.Common.WindowUnFullscreen,
  Events.Types.Common.WindowRestore,
]

// One window, so one copy of its state, shared by every component that asks.
const maximised = ref(false)
const fullscreen = ref(false)
let users = 0
let stops: (() => void)[] = []

const refresh = async () => {
  try {
    const [isFullscreen, isMaximised] = await Promise.all([Window.IsFullscreen(), Window.IsMaximised()])
    fullscreen.value = isFullscreen
    maximised.value = isMaximised
  } catch {
    // Outside the app window (e.g. a plain browser in dev) there is no state to read.
  }
}

/** Tracks whether the main window is maximised or fullscreen. */
export function useWindowState() {
  onMounted(() => {
    users++
    if (users > 1) {
      return
    }
    stops = changeEvents.map((name) => Events.On(name, () => void refresh()))
    void refresh()
  })

  onBeforeUnmount(() => {
    users--
    if (users > 0) {
      return
    }
    for (const stop of stops) {
      stop()
    }
    stops = []
  })

  return { maximised, fullscreen }
}

import { Events } from '@wailsio/runtime'

/**
 * Listens for an event from the Go backend and passes its data to the callback.
 * Returns a function that stops listening.
 */
export function onEvent<T = unknown>(name: string, callback: (data: T) => void): () => void {
  return Events.On(name, (event) => callback(event.data as T))
}

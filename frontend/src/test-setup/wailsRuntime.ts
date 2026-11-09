// @wailsio/runtime asks the Wails asset server for /wails/custom.js when it is
// imported. Tests have no asset server, so answer those requests with a 404
// instead of letting happy-dom log a refused connection for every import.
const realFetch = globalThis.fetch

if (realFetch) {
  globalThis.fetch = (input: RequestInfo | URL, init?: RequestInit) => {
    const url = input instanceof Request ? input.url : String(input)
    if (new URL(url, 'http://localhost').pathname.startsWith('/wails/')) {
      return Promise.resolve(new Response(null, { status: 404 }))
    }
    return realFetch(input, init)
  }
}

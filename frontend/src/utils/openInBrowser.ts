import * as systemProxy from 'bindings/vervet/internal/api/systemproxy'

/**
 * Opens an http(s) link in the user's browser. Use this, not the Wails
 * runtime's Browser.OpenURL: the backend refuses other schemes, so a link in a
 * document can't launch file://, smb:// or a custom URL handler. The backend
 * logs refused links.
 */
export function openInBrowser(url: string): void {
  void systemProxy.OpenURL(url)
}

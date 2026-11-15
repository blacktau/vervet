import * as systemProxy from 'bindings/vervet/internal/api/systemproxy'
import type { EmptyResult } from 'bindings/vervet/internal/api/models'
import { i18nGlobal } from '@/i18n'
import { useNotifier } from '@/utils/dialog'

/**
 * Opens an http(s) link in the user's browser. Use this, not the Wails
 * runtime's Browser.OpenURL: the backend refuses other schemes, so a link in a
 * document can't launch file://, smb:// or a custom URL handler.
 */
export async function openInBrowser(url: string): Promise<void> {
  reportRefusedLink(url, await systemProxy.OpenURL(url))
}

/** Tells the user when a link couldn't be opened, so they can copy it instead. */
export function reportRefusedLink(url: string, result: EmptyResult): void {
  if (result.isSuccess) {
    return
  }
  useNotifier().error(url, {
    title: i18nGlobal.t('errorTitles.openLink'),
    detail: result.errorDetail,
  })
}

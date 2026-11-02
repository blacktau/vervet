import { i18nGlobal } from '@/i18n'

/** The text to show for a failed proxy result. The Go side always sets one of these, but the types can't say so. */
export function resultErrorText(result: { errorCode?: string; errorDetail?: string }): string {
  return result.errorDetail || result.errorCode || i18nGlobal.t('errors.unknown_error')
}

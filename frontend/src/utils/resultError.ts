import { i18nGlobal } from '@/i18n'

// The Go side's catch-all: it says nothing the detail doesn't say better.
const UNKNOWN_ERROR = 'unknown_error'

type FailedResult = { errorCode?: string; errorDetail?: string }

/** A headline for an error code, for use where the detail is shown separately. */
export function errorCodeText(errorCode?: string): string {
  return i18nGlobal.t(`errors.${errorCode || UNKNOWN_ERROR}`)
}

/** The one line to show for a failed proxy result, where there's no room for a separate detail. */
export function resultErrorText({ errorCode, errorDetail }: FailedResult): string {
  if (errorCode && errorCode !== UNKNOWN_ERROR) {
    const key = `errors.${errorCode}`
    const translated = i18nGlobal.t(key)
    if (translated !== key) {
      return translated
    }
  }
  return errorDetail || errorCodeText()
}

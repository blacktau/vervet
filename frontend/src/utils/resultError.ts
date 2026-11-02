import { i18nGlobal } from '@/i18n'

// The Go side's catch-all: it says nothing the detail doesn't say better.
const UNKNOWN_ERROR = 'unknown_error'

type FailedResult = { errorCode?: string; errorDetail?: string }

function translateCode(errorCode?: string): string | undefined {
  if (!errorCode) {
    return undefined
  }
  const key = `errors.${errorCode}`
  const translated = i18nGlobal.t(key)
  return translated === key ? undefined : translated
}

/** A headline for an error code, for use where the detail is shown separately. */
export function errorCodeText(errorCode?: string): string {
  return translateCode(errorCode) ?? i18nGlobal.t(`errors.${UNKNOWN_ERROR}`)
}

/** The one line to show for a failed proxy result, where there's no room for a separate detail. */
export function resultErrorText({ errorCode, errorDetail }: FailedResult): string {
  const translated = errorCode === UNKNOWN_ERROR ? undefined : translateCode(errorCode)
  return translated || errorDetail || errorCodeText()
}

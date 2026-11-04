import { i18nGlobal } from '@/i18n'
import { useNotifier } from '@/utils/dialog'

// The Go side's catch-all: it says nothing the detail doesn't say better.
const UNKNOWN_ERROR = 'unknown_error'

export type FailedResult = { errorCode?: string; errorDetail?: string }

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

/** A short message for a failed proxy result, where a known code is enough and the detail would be noise. */
export function resultErrorText({ errorCode, errorDetail }: FailedResult): string {
  const translated = errorCode === UNKNOWN_ERROR ? undefined : translateCode(errorCode)
  return translated || errorDetail || errorCodeText()
}

/**
 * A message that leads with the real error, for failures that aren't database operations.
 * Go classifies errors partly by their text, so a code like query_timeout can mislead here.
 */
export function resultDetailText({ errorCode, errorDetail }: FailedResult): string {
  return errorDetail || errorCodeText(errorCode)
}

/** The whole message as text, for error states that render a string: the headline, then the detail. */
export function resultErrorLines({ errorCode, errorDetail }: FailedResult): string {
  const headline = errorCode === UNKNOWN_ERROR ? undefined : translateCode(errorCode)
  if (!headline) {
    return errorDetail || errorCodeText()
  }
  return errorDetail && errorDetail !== headline ? `${headline}\n${errorDetail}` : headline
}

/** Notifies of a failure that isn't a database operation, leading with the real error. See resultDetailText. */
export function notifyResultDetail(result: FailedResult, title?: string): void {
  useNotifier().error(resultDetailText(result), { title })
}

/** Notifies of a failed proxy result, keeping the detail beneath the headline. */
export function notifyResultError(result: FailedResult, title?: string): void {
  useNotifier().error(errorCodeText(result.errorCode), { title, detail: result.errorDetail })
}

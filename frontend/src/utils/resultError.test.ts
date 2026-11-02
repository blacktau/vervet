import { describe, expect, it } from 'vitest'
import { i18nGlobal } from '@/i18n'
import { errorCodeText, resultErrorText } from './resultError'

const generic = () => i18nGlobal.t('errors.unknown_error')
const notAuthorised = () => i18nGlobal.t('errors.not_authorized')

describe('errorCodeText', () => {
  it('translates a code', () => {
    expect(errorCodeText('not_authorized')).toBe(notAuthorised())
  })

  it('uses the generic message when there is no code', () => {
    expect(errorCodeText(undefined)).toBe(generic())
    expect(errorCodeText('')).toBe(generic())
  })
})

describe('resultErrorText', () => {
  it('prefers the translation of a known code', () => {
    expect(resultErrorText({ errorCode: 'not_authorized', errorDetail: 'raw' })).toBe(notAuthorised())
  })

  it("shows the detail for the Go side's catch-all code", () => {
    expect(resultErrorText({ errorCode: 'unknown_error', errorDetail: 'E11000 duplicate key' })).toBe(
      'E11000 duplicate key',
    )
  })

  it('shows the detail when there is no code', () => {
    expect(resultErrorText({ errorDetail: 'E11000 duplicate key' })).toBe('E11000 duplicate key')
  })

  it('shows the detail when the code has no translation', () => {
    expect(resultErrorText({ errorCode: 'made_up', errorDetail: 'raw' })).toBe('raw')
  })

  it('never shows a raw code', () => {
    expect(resultErrorText({ errorCode: 'unknown_error' })).toBe(generic())
    expect(resultErrorText({ errorCode: 'made_up' })).toBe(generic())
  })

  it('uses the generic message when the result says nothing', () => {
    expect(resultErrorText({})).toBe(generic())
  })
})

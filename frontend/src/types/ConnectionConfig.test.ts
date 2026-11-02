import { describe, expect, it } from 'vitest'
import { toAuthMethod } from './ConnectionConfig'

describe('toAuthMethod', () => {
  it('keeps a known method', () => {
    expect(toAuthMethod('oidc')).toBe('oidc')
  })

  it('defaults an empty value to password', () => {
    expect(toAuthMethod('')).toBe('password')
  })

  it('keeps an unrecognised value, so saving does not overwrite it', () => {
    expect(toAuthMethod('kerberos')).toBe('kerberos')
  })
})

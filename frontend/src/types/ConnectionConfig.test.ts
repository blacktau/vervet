import { describe, expect, it } from 'vitest'
import { toAuthMethod } from './ConnectionConfig'

describe('toAuthMethod', () => {
  it('keeps a known method', () => {
    expect(toAuthMethod('oidc')).toBe('oidc')
  })

  it('falls back to password for an empty or unknown value', () => {
    expect(toAuthMethod('')).toBe('password')
    expect(toAuthMethod('kerberos')).toBe('password')
  })
})

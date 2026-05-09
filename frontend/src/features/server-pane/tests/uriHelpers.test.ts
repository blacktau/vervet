import { describe, expect, test } from 'vitest'
import { detectAuthFromUri, getUriHost } from '@/features/server-pane/connectionStrings.ts'

describe('detectAuthFromUri', () => {
  test('returns password for plain URI', () => {
    const result = detectAuthFromUri('mongodb://localhost:27017')
    expect(result).toEqual({ authMethod: 'password', uri: 'mongodb://localhost:27017' })
  })

  test('detects OIDC and strips authMechanism + authMechanismProperties', () => {
    const result = detectAuthFromUri(
      'mongodb://host/?authMechanism=MONGODB-OIDC&authMechanismProperties=PROVIDER_NAME:aws',
    )
    expect(result.authMethod).toBe('oidc')
    expect(result.uri.toLowerCase()).not.toContain('authmechanism=')
    expect(result.uri.toLowerCase()).not.toContain('authmechanismproperties=')
  })

  test('detects x509 without modifying URI', () => {
    const uri = 'mongodb://host/?authMechanism=MONGODB-X509'
    const result = detectAuthFromUri(uri)
    expect(result).toEqual({ authMethod: 'x509', uri })
  })

  test('detects AWS without modifying URI', () => {
    const uri = 'mongodb://host/?authMechanism=MONGODB-AWS'
    const result = detectAuthFromUri(uri)
    expect(result).toEqual({ authMethod: 'aws', uri })
  })

  test('case-insensitive match', () => {
    const result = detectAuthFromUri('mongodb://host/?AuthMechanism=mongodb-oidc')
    expect(result.authMethod).toBe('oidc')
  })

  test('leaves leading question mark clean after OIDC stripping', () => {
    const result = detectAuthFromUri(
      'mongodb://host/?authMechanism=MONGODB-OIDC',
    )
    expect(result.uri).toBe('mongodb://host/')
  })
})

describe('getUriHost', () => {
  test('returns host:port for plain URI', () => {
    expect(getUriHost('mongodb://localhost:27017')).toBe('localhost:27017')
  })

  test('returns host only when port absent', () => {
    expect(getUriHost('mongodb://example.com')).toBe('example.com')
  })

  test('returns first host for multi-host URI', () => {
    expect(getUriHost('mongodb://a.example.com,b.example.com:27018')).toBe('a.example.com')
  })

  test('returns srv host', () => {
    expect(getUriHost('mongodb+srv://cluster0.mongodb.net')).toBe('cluster0.mongodb.net')
  })

  test('strips userinfo', () => {
    expect(getUriHost('mongodb://user:pw@host:27017')).toBe('host:27017')
  })

  test('returns empty string for invalid URI', () => {
    expect(getUriHost('not-a-uri')).toBe('')
  })

  test('returns empty string for empty input', () => {
    expect(getUriHost('')).toBe('')
  })
})

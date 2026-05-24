import { describe, expect, it } from 'vitest'
import {
  parseScram,
  serialiseScram,
  parseX509,
  serialiseX509,
  parseAws,
  serialiseAws,
  parseGssapi,
  serialiseGssapi,
  parsePlain,
  serialisePlain,
} from '../authUri'

describe('SCRAM', () => {
  it('round-trips username + password + authSource + mechanism', () => {
    const uri = 'mongodb://alice:p%40ss@host/?authSource=admin&authMechanism=SCRAM-SHA-256'
    const parsed = parseScram(uri)
    expect(parsed).toEqual({
      username: 'alice',
      password: 'p@ss',
      authSource: 'admin',
      mechanism: 'SCRAM-SHA-256',
    })
    expect(serialiseScram(uri, parsed)).toBe(uri)
  })

  it('omits empty fields on serialise', () => {
    expect(
      serialiseScram('mongodb://host/', { username: '', password: '', mechanism: 'auto' }),
    ).toBe('mongodb://host/')
  })

  it('URL-encodes special characters in credentials', () => {
    const out = serialiseScram('mongodb://host/', {
      username: 'a@b',
      password: 'p:s/w',
      mechanism: 'auto',
    })
    expect(out).toBe('mongodb://a%40b:p%3As%2Fw@host/')
  })
})

describe('X.509', () => {
  it('round-trips cert path, passphrase, username override, authSource', () => {
    const uri =
      'mongodb://CN%3Dme@host/?authMechanism=MONGODB-X509&authSource=$external' +
      '&tlsCertificateKeyFile=%2Ftmp%2Fclient.pem&tlsCertificateKeyFilePassword=secret'
    const parsed = parseX509(uri)
    expect(parsed.certFile).toBe('/tmp/client.pem')
    expect(parsed.certPassphrase).toBe('secret')
    expect(parsed.usernameOverride).toBe('CN=me')
    expect(parsed.authSource).toBe('$external')
    expect(serialiseX509(uri, parsed)).toBe(uri)
  })
})

describe('AWS', () => {
  it('round-trips access key, secret, session token', () => {
    const uri =
      'mongodb://AKIA:secret@host/?authMechanism=MONGODB-AWS' +
      '&authMechanismProperties=AWS_SESSION_TOKEN:tok123'
    const parsed = parseAws(uri)
    expect(parsed).toEqual({
      accessKeyId: 'AKIA',
      secretAccessKey: 'secret',
      sessionToken: 'tok123',
    })
    expect(serialiseAws(uri, parsed)).toBe(uri)
  })

  it('omits sessionToken when blank', () => {
    const out = serialiseAws('mongodb://host/', { accessKeyId: 'AKIA', secretAccessKey: 'sec' })
    expect(out).toBe('mongodb://AKIA:sec@host/?authMechanism=MONGODB-AWS')
  })
})

describe('GSSAPI', () => {
  it('round-trips principal + service properties', () => {
    const uri =
      'mongodb://alice%40REALM@host/?authMechanism=GSSAPI' +
      '&authMechanismProperties=SERVICE_NAME:mongodb,CANONICALIZE_HOST_NAME:forward,SERVICE_REALM:REALM'
    const parsed = parseGssapi(uri)
    expect(parsed.principal).toBe('alice@REALM')
    expect(parsed.serviceName).toBe('mongodb')
    expect(parsed.canonicalize).toBe('forward')
    expect(parsed.serviceRealm).toBe('REALM')
    expect(serialiseGssapi(uri, parsed)).toBe(uri)
  })
})

describe('PLAIN', () => {
  it('round-trips username + password + authSource=$external', () => {
    const uri = 'mongodb://alice:pw@host/?authMechanism=PLAIN&authSource=$external'
    const parsed = parsePlain(uri)
    expect(parsed).toEqual({ username: 'alice', password: 'pw', authSource: '$external' })
    expect(serialisePlain(uri, parsed)).toBe(uri)
  })
})

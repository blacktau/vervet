export interface ScramAuth {
  username: string
  password: string
  authSource?: string
  mechanism?: 'auto' | 'SCRAM-SHA-1' | 'SCRAM-SHA-256'
}

export interface X509Auth {
  certFile: string
  certPassphrase?: string
  usernameOverride?: string
  authSource?: string
}

export interface AwsAuth {
  accessKeyId: string
  secretAccessKey: string
  sessionToken?: string
}

export interface GssapiAuth {
  principal: string
  serviceName?: string
  canonicalize?: 'none' | 'forward' | 'forwardAndReverse'
  serviceRealm?: string
  password?: string
}

export interface PlainAuth {
  username: string
  password: string
  authSource?: string
}

interface UriParts {
  scheme: string
  userinfo: string
  hostAndPath: string
  query: Record<string, string>
}

function splitUri(uri: string): UriParts {
  const schemeMatch = uri.match(/^(mongodb(?:\+srv)?:\/\/)/)
  const scheme = schemeMatch ? schemeMatch[1]! : 'mongodb://'
  const rest = uri.substring(scheme.length)

  const queryIdx = rest.indexOf('?')
  const queryStr = queryIdx === -1 ? '' : rest.substring(queryIdx + 1)
  const beforeQuery = queryIdx === -1 ? rest : rest.substring(0, queryIdx)

  const atIdx = beforeQuery.lastIndexOf('@')
  const userinfo = atIdx === -1 ? '' : beforeQuery.substring(0, atIdx)
  const hostAndPath = atIdx === -1 ? beforeQuery : beforeQuery.substring(atIdx + 1)

  const query: Record<string, string> = {}
  if (queryStr.length > 0) {
    for (const part of queryStr.split('&')) {
      const eq = part.indexOf('=')
      if (eq === -1) {
        continue
      }
      query[part.substring(0, eq)] = part.substring(eq + 1)
    }
  }

  return { scheme, userinfo, hostAndPath, query }
}

function joinUri(parts: UriParts): string {
  const userinfo = parts.userinfo ? `${parts.userinfo}@` : ''
  const keys = Object.keys(parts.query)
  // Driver's connstring.Parse requires "/" between host and "?". Insert it
  // when the host segment has no path of its own.
  const hostAndPath =
    keys.length > 0 && parts.hostAndPath.indexOf('/') === -1
      ? `${parts.hostAndPath}/`
      : parts.hostAndPath
  const query =
    keys.length === 0 ? '' : `?${keys.map((k) => `${k}=${parts.query[k]}`).join('&')}`
  return `${parts.scheme}${userinfo}${hostAndPath}${query}`
}

function encodeUserinfo(user: string, password?: string): string {
  const u = encodeURIComponent(user)
  if (password === undefined || password === '') {
    return u
  }
  return `${u}:${encodeURIComponent(password)}`
}

function decodeUserinfo(userinfo: string): { username: string; password: string } {
  if (userinfo === '') {
    return { username: '', password: '' }
  }
  const colon = userinfo.indexOf(':')
  if (colon === -1) {
    return { username: decodeURIComponent(userinfo), password: '' }
  }
  return {
    username: decodeURIComponent(userinfo.substring(0, colon)),
    password: decodeURIComponent(userinfo.substring(colon + 1)),
  }
}

function parseMechProps(value: string): Record<string, string> {
  const out: Record<string, string> = {}
  if (!value) {
    return out
  }
  for (const pair of value.split(',')) {
    const colon = pair.indexOf(':')
    if (colon === -1) {
      continue
    }
    out[decodeURIComponent(pair.substring(0, colon))] = decodeURIComponent(
      pair.substring(colon + 1),
    )
  }
  return out
}

function serialiseMechProps(props: Record<string, string>): string {
  const order = ['SERVICE_NAME', 'CANONICALIZE_HOST_NAME', 'SERVICE_REALM', 'AWS_SESSION_TOKEN']
  const seen = new Set<string>()
  const out: string[] = []
  for (const k of order) {
    if (props[k] !== undefined && props[k] !== '') {
      out.push(`${k}:${encodeURIComponent(props[k]!)}`)
      seen.add(k)
    }
  }
  for (const k of Object.keys(props)) {
    if (seen.has(k) || props[k] === '' || props[k] === undefined) {
      continue
    }
    out.push(`${encodeURIComponent(k)}:${encodeURIComponent(props[k]!)}`)
  }
  return out.join(',')
}

// ----- SCRAM -----

export function parseScram(uri: string): ScramAuth {
  const parts = splitUri(uri)
  const { username, password } = decodeUserinfo(parts.userinfo)
  const mech = parts.query['authMechanism']
  return {
    username,
    password,
    authSource: parts.query['authSource']
      ? decodeURIComponent(parts.query['authSource'])
      : undefined,
    mechanism: mech === 'SCRAM-SHA-1' || mech === 'SCRAM-SHA-256' ? mech : 'auto',
  }
}

export function serialiseScram(uri: string, fields: ScramAuth): string {
  const parts = splitUri(uri)
  parts.userinfo = fields.username ? encodeUserinfo(fields.username, fields.password) : ''
  delete parts.query['authMechanismProperties']
  delete parts.query['tlsCertificateKeyFile']
  delete parts.query['tlsCertificateKeyFilePassword']
  if (fields.mechanism === 'SCRAM-SHA-1' || fields.mechanism === 'SCRAM-SHA-256') {
    parts.query['authMechanism'] = fields.mechanism
  } else {
    delete parts.query['authMechanism']
  }
  if (fields.authSource) {
    parts.query['authSource'] = fields.authSource
  } else {
    delete parts.query['authSource']
  }
  return joinUri(parts)
}

// ----- X.509 -----

export function parseX509(uri: string): X509Auth {
  const parts = splitUri(uri)
  const userinfo = parts.userinfo ? decodeUserinfo(parts.userinfo).username : ''
  return {
    certFile: parts.query['tlsCertificateKeyFile']
      ? decodeURIComponent(parts.query['tlsCertificateKeyFile'])
      : '',
    certPassphrase: parts.query['tlsCertificateKeyFilePassword']
      ? decodeURIComponent(parts.query['tlsCertificateKeyFilePassword'])
      : undefined,
    usernameOverride: userinfo || undefined,
    authSource: parts.query['authSource']
      ? decodeURIComponent(parts.query['authSource'])
      : '$external',
  }
}

export function serialiseX509(uri: string, fields: X509Auth): string {
  const parts = splitUri(uri)
  parts.userinfo = fields.usernameOverride ? encodeUserinfo(fields.usernameOverride) : ''
  delete parts.query['authMechanismProperties']
  parts.query['authMechanism'] = 'MONGODB-X509'
  parts.query['authSource'] = fields.authSource ?? '$external'
  if (fields.certFile) {
    parts.query['tlsCertificateKeyFile'] = encodeURIComponent(fields.certFile)
  } else {
    delete parts.query['tlsCertificateKeyFile']
  }
  if (fields.certPassphrase) {
    parts.query['tlsCertificateKeyFilePassword'] = encodeURIComponent(fields.certPassphrase)
  } else {
    delete parts.query['tlsCertificateKeyFilePassword']
  }
  return joinUri(parts)
}

// ----- AWS -----

export function parseAws(uri: string): AwsAuth {
  const parts = splitUri(uri)
  const { username, password } = decodeUserinfo(parts.userinfo)
  const props = parseMechProps(parts.query['authMechanismProperties'] ?? '')
  return {
    accessKeyId: username,
    secretAccessKey: password,
    sessionToken: props['AWS_SESSION_TOKEN'] || undefined,
  }
}

export function serialiseAws(uri: string, fields: AwsAuth): string {
  const parts = splitUri(uri)
  parts.userinfo = fields.accessKeyId
    ? encodeUserinfo(fields.accessKeyId, fields.secretAccessKey)
    : ''
  delete parts.query['authSource']
  delete parts.query['tlsCertificateKeyFile']
  delete parts.query['tlsCertificateKeyFilePassword']
  parts.query['authMechanism'] = 'MONGODB-AWS'
  const props: Record<string, string> = {}
  if (fields.sessionToken) {
    props['AWS_SESSION_TOKEN'] = fields.sessionToken
  }
  const propsStr = serialiseMechProps(props)
  if (propsStr) {
    parts.query['authMechanismProperties'] = propsStr
  } else {
    delete parts.query['authMechanismProperties']
  }
  return joinUri(parts)
}

// ----- GSSAPI -----

export function parseGssapi(uri: string): GssapiAuth {
  const parts = splitUri(uri)
  const { username, password } = decodeUserinfo(parts.userinfo)
  const props = parseMechProps(parts.query['authMechanismProperties'] ?? '')
  const canonical = props['CANONICALIZE_HOST_NAME']
  return {
    principal: username,
    password: password || undefined,
    serviceName: props['SERVICE_NAME'] || undefined,
    canonicalize:
      canonical === 'forward' || canonical === 'forwardAndReverse' || canonical === 'none'
        ? canonical
        : undefined,
    serviceRealm: props['SERVICE_REALM'] || undefined,
  }
}

export function serialiseGssapi(uri: string, fields: GssapiAuth): string {
  const parts = splitUri(uri)
  parts.userinfo = fields.principal
    ? encodeUserinfo(fields.principal, fields.password)
    : ''
  delete parts.query['authSource']
  delete parts.query['tlsCertificateKeyFile']
  delete parts.query['tlsCertificateKeyFilePassword']
  parts.query['authMechanism'] = 'GSSAPI'
  const props: Record<string, string> = {}
  if (fields.serviceName) {
    props['SERVICE_NAME'] = fields.serviceName
  }
  if (fields.canonicalize) {
    props['CANONICALIZE_HOST_NAME'] = fields.canonicalize
  }
  if (fields.serviceRealm) {
    props['SERVICE_REALM'] = fields.serviceRealm
  }
  const propsStr = serialiseMechProps(props)
  if (propsStr) {
    parts.query['authMechanismProperties'] = propsStr
  } else {
    delete parts.query['authMechanismProperties']
  }
  return joinUri(parts)
}

// ----- PLAIN -----

export function parsePlain(uri: string): PlainAuth {
  const parts = splitUri(uri)
  const { username, password } = decodeUserinfo(parts.userinfo)
  return {
    username,
    password,
    authSource: parts.query['authSource']
      ? decodeURIComponent(parts.query['authSource'])
      : '$external',
  }
}

export function serialisePlain(uri: string, fields: PlainAuth): string {
  const parts = splitUri(uri)
  parts.userinfo = fields.username ? encodeUserinfo(fields.username, fields.password) : ''
  delete parts.query['authMechanismProperties']
  delete parts.query['tlsCertificateKeyFile']
  delete parts.query['tlsCertificateKeyFilePassword']
  parts.query['authMechanism'] = 'PLAIN'
  parts.query['authSource'] = fields.authSource ?? '$external'
  return joinUri(parts)
}

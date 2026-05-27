import { i18nGlobal } from '@/i18n'
import type { AuthMethod, OIDCConfig } from '@/types/ConnectionConfig'

interface OptionValidator {
  v: (value?: string) => boolean
  m: string
}

const uriOptions: Record<string, OptionValidator | null> = {
  appname: { v: validateAppName, m: 'uriParser.appNameTooLong' },
  authMechanism: { v: validateAuthMechanism, m: 'uriParser.invalidAuthMechanism' },
  authMechanismProperties: {
    v: validateAuthMechanismProps,
    m: 'uriParser.invalidAuthMechanismProps',
  },
  authSource: { v: validateNonEmptyString, m: 'uriParser.authSourceRequired' },
  compressors: { v: validateCompressors, m: 'uriParser.invalidCompressors' },
  connectTimeoutMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  directConnection: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  heartbeatFrequencyMS: { v: validateTimeout, m: 'uriParse.invalidTimeout' },
  journal: { v: validateBoolean, m: 'uriParse.invalidBoolean' },
  loadBalanced: null,
  localThresholdMS: { v: validatePositiveFloat, m: 'uriParser.invalidPositiveFloat' },
  maxIdleTimeMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  maxPoolSize: { v: validateNonNegativeInteger, m: 'uriParser.invalidNonNegativeInteger' },
  maxConnecting: { v: validatePositiveInteger, m: 'uriParser.invalidPositiveInteger' },
  maxStalenessSeconds: { v: validateMaxStaleness, m: 'uriParser.invalidMaxStaleness' },
  minPoolSize: { v: validateNonNegativeInteger, m: 'uriParser.invalidNonNegativeInteger' },
  proxyHost: null,
  proxyPort: null,
  proxyUsername: null,
  proxyPassword: null,
  readConcernLevel: null,
  readPreference: { v: validateReadPreferenceMode, m: 'uriParser.invalidReadPreferenceMode' },
  readPreferenceTags: null,
  replicaSet: null,
  retryReads: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  retryWrites: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  serverMonitoringMode: {
    v: validateServerMonitoringMode,
    m: 'uriParser.invalidServerMonitoringMode',
  },
  serverSelectionTimeoutMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  serverSelectionTryOnce: null,
  socketTimeoutMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  srvMaxHosts: { v: validateNonNegativeInteger, m: 'uriParser.invalidNonNegativeInteger' },
  srvServiceName: null,
  ssl: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  tls: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  tlsAllowInvalidCertificates: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  tlsAllowInvalidHostnames: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  tlsCAFile: null,
  tlsCertificateKeyFile: null,
  tlsCertificateKeyFilePassword: null,
  tlsDisableCertificateRevocationCheck: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  tlsDisableOCSPEndpointCheck: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  tlsInsecure: { v: validateBoolean, m: 'uriParser.invalidBoolean' },
  timeoutMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  w: { v: validateWriteConcern, m: 'uriParser.invalidWriteConcern' },
  waitQueueTimeoutMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  wTimeoutMS: { v: validateTimeout, m: 'uriParser.invalidTimeout' },
  zlibCompressionLevel: {
    v: validateZlibCompressionLevel,
    m: 'uriParser.invalidNonNegativeInteger',
  },
}

const implicitTlsInsecureOptions = [
  'tlsAllowInvalidCertificates',
  'tlsAllowInvalidHostnames',
  'tlsDisableOCSPEndpointCheck',
]

const Scheme = {
  MONGODB: 'mongodb://',
  MONGODB_SHARD: 'mongodb+srv://',
}

const AuthMechanism = [
  'GSSAPI',
  'MONGODB-X509',
  'MONGODB-AWS',
  'MONGODB-OIDC',
  'PLAIN',
  'SCRAM-SHA-1',
  'SCRAM-SHA-256',
]

const SupportedCompressors = ['snappy', 'zlib', 'zstd']

const ReadPreferenceMode = [
  'primary',
  'primaryPreferred',
  'secondary',
  'secondaryPreferred',
  'nearest',
]

type ParseResult<T> = {
  success: boolean
  error?: string
  data?: T
}

type UriData = {
  nodelist: Address[]
  username?: string
  password?: string
  database: string | undefined
  collection: string | undefined
  options?: UriOptions
  isSrv: boolean
  fqdn?: string
}

type UriOptions = Record<string, string | string[] | Record<string, string> | boolean | number>

const badDatabaseChars = /[\/ "$]/

export const parseUri = (uri: string): ParseResult<UriData> => {
  const result = parseAndValidateUri(uri)
  if (!result.success) {
    return result
  }

  return {
    success: true,
    data: result.data!,
  }
}

const parseAndValidateUri = (uri: string): ParseResult<UriData> => {
  let isSrv = false
  let schemeLess = ''
  if (uri.startsWith(Scheme.MONGODB)) {
    isSrv = false
    schemeLess = uri.substring(Scheme.MONGODB.length)
  } else if (uri.startsWith(Scheme.MONGODB_SHARD)) {
    isSrv = true
    schemeLess = uri.substring(Scheme.MONGODB_SHARD.length)
  } else {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.invalidScheme'),
    }
  }

  if (schemeLess.length === 0) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.emptyUri'),
    }
  }

  const [hostAndDatabase, query] = chopFirst(schemeLess, '?')
  const trimmedHostAndDatabase = hostAndDatabase!.endsWith('/')
    ? hostAndDatabase!.slice(0, -1)
    : hostAndDatabase!
  let [host, database] = chopAtLast(trimmedHostAndDatabase, '/')

  let collection: string | undefined = undefined

  if (database != null) {
    if (database.length === 0) {
      database = undefined
    } else {
      database = decodeURIComponent(database)
      if (database.indexOf('.') === -1) {
        ;[database, collection] = chopAtLast(database, '.')
        if (badDatabaseChars.test(database!)) {
          return {
            success: false,
            error: i18nGlobal.t('uriParser.invalidDatabaseName'),
          }
        }
      }
    }
  }

  const queryResult = parseAndValidateOptions(query!)

  if (!queryResult.success) {
    return {
      success: false,
      error: queryResult.error,
    }
  }

  const options = queryResult.data!

  let username: string | undefined
  let password: string | undefined

  if (host!.indexOf('@') >= 0) {
    const userinfo = host!.substring(0, host!.lastIndexOf('@'))
    host = host!.substring(host!.lastIndexOf('@') + 1)
    const userResult = parseUserInfo(userinfo)

    if (!userResult.success) {
      return {
        success: false,
        error: userResult.error,
      }
    }

    username = userResult.data!.username
    password = userResult.data!.password
  }

  if (host!.indexOf('/') >= 0) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.invalidHostSlash', { host: host! }),
    }
  }

  const fqdn: string | undefined = undefined
  let nodes: Address[]
  const srvMaxHosts = options['srvMaxHosts']
  if (isSrv) {
    if (
      options['directConnection'] &&
      (options['directConnection'] as string).toLowerCase() === 'true'
    ) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.srvDirectConnection'),
      }
    }

    const nodeResult = splitHosts(host!, null)
    if (!nodeResult.success) {
      return {
        success: false,
        error: nodeResult.error,
      }
    }

    nodes = nodeResult.data!

    if (nodes.length !== 1) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.srvMultipleHosts'),
      }
    }

    if (nodes[0]?.port !== null) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.srvPortSpecified'),
      }
    }
  } else if (!isSrv && options['srvServiceName']) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.srvServiceNameNotSrv'),
    }
  } else if (!isSrv && srvMaxHosts) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.srvMaxHostsNotSrv'),
    }
  } else {
    const nodeResult2 = splitHosts(host!, null)
    if (!nodeResult2.success) {
      return {
        success: false,
        error: nodeResult2.error,
      }
    }
    nodes = nodeResult2.data!
  }

  if (
    nodes.length > 1 &&
    options['directConnection'] &&
    (options['directConnection'] as string).toLowerCase() === 'true'
  ) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.directConnectionMultipleHosts'),
    }
  }

  if (options['loadBalanced'] && (options['loadBalanced'] as string).toLowerCase() === 'true') {
    if (nodes.length > 1) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.loadBalancedMultipleHosts'),
      }
    }

    if (
      options['directConnection'] &&
      (options['directConnection'] as string).toLowerCase() === 'true'
    ) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.loadBalancedDirectConnection'),
      }
    }

    if (options['replicaSet'] && (options['replicaSet'] as string).toLowerCase() === 'true') {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.loadBalancedReplicaSet'),
      }
    }
  }

  return {
    success: true,
    data: {
      nodelist: nodes,
      username,
      password,
      database,
      collection,
      options: Object.keys(options).length === 0 ? undefined : options,
      isSrv,
      fqdn,
    },
  }
}

type Address = {
  host: string
  port?: number | undefined | null
}

function chopAtLast(target: string, separator: string) {
  const lastIndex = target.lastIndexOf(separator)
  if (lastIndex === -1) {
    return [target, undefined]
  }
  const leader = target.substring(0, lastIndex)
  const tail = target.substring(lastIndex + separator.length)
  return [leader, tail] as [string, string | undefined]
}

function chopFirst(target: string, separator: string) {
  const firstIndex = target.indexOf(separator)
  if (firstIndex === -1) {
    return [target, undefined]
  }
  const tail = target.substring(firstIndex + separator.length)
  const leader = target.substring(0, firstIndex)
  return [leader, tail] as [string, string | undefined]
}

function splitHosts(hosts: string, defaultPort: number | undefined | null): ParseResult<Address[]> {
  const hostEntries = hosts.split(',')
  if (hostEntries.length === 0) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.emptyHostList'),
    }
  }

  const parsedAddresses: Address[] = []
  for (const hostEntry of hostEntries) {
    const [host, portNumber] = chopAtLast(hostEntry.trim(), ':')
    if (host!.indexOf(':') >= 0 && host!.indexOf(']') < 0) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.invalidHostEntry', { host: hostEntry }),
      }
    }

    if (portNumber) {
      const port = Number.parseInt(portNumber)
      if (port > 0 && port < 65535) {
        parsedAddresses.push({ host: host!, port: port })
      } else {
        return {
          success: false,
          error: i18nGlobal.t('uriParser.invalidPort', { port: port }),
        }
      }
    } else {
      parsedAddresses.push({ host: host!, port: defaultPort })
    }
  }

  return {
    success: true,
    data: parsedAddresses,
  }
}

function parseUserInfo(userInfo: string): ParseResult<{ username?: string; password?: string }> {
  if (userInfo.indexOf('/') >= 0) {
    console.warn(`UserInfo contains escaped slash: ${userInfo}`)
  }
  if (
    userInfo.indexOf('@') >= 0 ||
    (userInfo.match(/:/g) || []).length > 1 ||
    unquotedPercent(userInfo)
  ) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.invalidUserInfo'),
    }
  }

  const [username, password] = chopAtLast(userInfo, ':')

  if (username == null) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.emptyUsername'),
    }
  }

  if ((username!.match(/\//g) || []).length > 0) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.unescapedUsername'),
    }
  }

  if (password && (password.match(/\//g) || []).length > 0) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.unescapedPassword'),
    }
  }

  return {
    success: true,
    data: {
      username: decodeURIComponent(username),
      password: password == null ? password : decodeURIComponent(password),
    },
  }
}

function unquotedPercent(s: string) {
  try {
    decodeURIComponent(s)
  } catch {
    return true
  }
  return false
}

const parseAndValidateOptions = (query: string): ParseResult<UriOptions> => {
  if (!query || query.length === 0) {
    return {
      success: true,
      data: {},
    }
  }

  const ampIdx = query.indexOf('&')
  const semicolonIdx = query.indexOf(';')
  let options = {} as UriOptions

  if (ampIdx >= 0 && semicolonIdx >= 0) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.invalidQueryMixingSeparators'),
    }
  } else if (ampIdx >= 0) {
    const optionResult = parseOptions(query, '&')
    if (!optionResult.success) {
      return optionResult
    }
    options = optionResult.data!
  } else if (semicolonIdx >= 0) {
    const optionResult = parseOptions(query, ';')
    if (!optionResult.success) {
      return optionResult
    }
    options = optionResult.data!
  } else if (query.indexOf('=') != 0) {
    const optionResult = parseOptions(query, undefined)
    if (!optionResult.success) {
      return optionResult
    }
    options = optionResult.data!
  } else {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.invalidQueryFormat'),
    }
  }

  const result = handleSecurityOptions(options)
  if (!result.success) {
    return result
  }

  options = result.data!

  if (options['authSource'] === '') {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.authSourceRequired'),
    }
  }

  // validate options
  for (const key in options) {
    if (!(key in uriOptions)) {
      continue
    }

    if (uriOptions[key] == null) {
      continue
    }

    // skip the tags
    if (Array.isArray(options[key])) {
      continue
    }

    if (typeof options[key] === 'object') {
      continue
    }

    if (!uriOptions[key]?.v(options[key]?.toString())) {
      return {
        success: false,
        error: i18nGlobal.t(uriOptions[key]?.m, { key: key, value: options[key] }),
      }
    }
  }

  return {
    success: true,
    data: options,
  }
}

const handleSecurityOptions = (options: UriOptions) => {
  if (options['tlsInsecure']) {
    for (const option of implicitTlsInsecureOptions) {
      if (option in options) {
        return {
          success: false,
          error: i18nGlobal.t('uriParser.conflictingOptions', {
            option1: 'tlsInsecure',
            option2: option,
          }),
        }
      }
    }
  }

  if (options['tlsAllowInvalidCertificates'] && options['tlsDisableOCSPEndpointCheck']) {
    return {
      success: false,
      error: i18nGlobal.t('uriParser.conflictingOptions', {
        option1: 'tlsAllowInvalidCertificates',
        option2: 'tlsDisableOCSPEndpointCheck',
      }),
    }
  }

  if (options['tls'] && options['ssl']) {
    if (options['tls'] !== options['ssl']) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.tlsAndSslConflict'),
      }
    }
  }

  return {
    success: true,
    data: options,
  }
}

const parseOptions = (query: string, separator?: string): ParseResult<UriOptions> => {
  const options = {} as UriOptions

  const parts = separator ? query.split(separator) : [query]

  for (const part of parts) {
    const optionParts = part.split('=')
    if (optionParts.length !== 2) {
      return {
        success: false,
        error: i18nGlobal.t('uriParser.invalidQueryOption', { option: part }),
      }
    }
    const key = optionParts[0]
    const value = optionParts[1]
    const normalizedKey = normalizeKey(key!)
    if (key!.toLowerCase() === 'readpreferencetags') {
      if (isDuplicateOption(normalizedKey, options)) {
        options[normalizedKey] = [...(options[normalizedKey] as string[]), value!]
      }

      options[normalizedKey] = [value!]
    } else {
      if (isDuplicateOption(normalizedKey, options)) {
        console.warn(`Duplicate option '${normalizedKey}' found in query string`)
      }

      if (key!.toLowerCase() === 'authmechanismproperties') {
        const authMechProps: Record<string, string> = {}
        const pairs = value!.split(',')
        for (const pair of pairs) {
          const [key, value] = chopFirst(pair, ':')
          authMechProps[decodeURIComponent(key!)] = decodeURIComponent(value!)
        }
        options[normalizedKey] = authMechProps
      } else {
        if (value!.toLowerCase() === 'true' || value!.toLowerCase() === 'false') {
          options[normalizedKey] = value!.toLowerCase() === 'true'
        } else if (value!.trim().match(/^[0-9]+$/)) {
          options[normalizedKey] = Number.parseInt(value!)
        } else if (value!.trim().match(/^[0-9.]+$/)) {
          options[normalizedKey] = Number.parseFloat(value!)
        } else {
          options[normalizedKey] = decodeURIComponent(value!)
        }
      }
    }
  }

  return {
    success: true,
    data: options,
  }
}

const normalizeKey = (key: string) => {
  const toCheck = key.toLowerCase()
  for (const validKey in uriOptions) {
    if (validKey.toLowerCase() === toCheck) {
      return validKey
    }
  }
  return key
}

const isDuplicateOption = (key: string, options: UriOptions) => {
  return key in options
}

function validateAppName(appName?: string) {
  if (!appName) {
    return true
  }

  return new TextEncoder().encode(appName).length <= 128
}

function validateAuthMechanism(authMechanism?: string) {
  if (!authMechanism) {
    return false
  }

  const upper = authMechanism.toUpperCase()
  return AuthMechanism.some((m) => m.toUpperCase() === upper)
}

// MongoDB write-concern value: a non-negative integer, the literal "majority",
// or a custom write-concern tag set name (any non-empty string).
function validateWriteConcern(value?: string) {
  if (!value) {
    return false
  }
  if (value.trim().match(/^[0-9]+$/)) {
    return Number.parseInt(value) >= 0
  }
  return value.length > 0
}

function validateAuthMechanismProps(authMechanismProps?: string) {
  if (!authMechanismProps) {
    return true
  }

  return true
}

function validateBoolean(value?: string) {
  if (!value) {
    return false
  }
  return value.toLowerCase() === 'true' || value.toLowerCase() === 'false'
}

function validatePositiveInteger(value?: string) {
  if (!value) {
    return false
  }

  try {
    const n = Number.parseInt(value)
    return n > 0
  } catch {
    return false
  }
}

function validateNonNegativeInteger(value?: string) {
  if (!value) {
    return false
  }

  try {
    const n = Number.parseInt(value)
    return n >= 0
  } catch {
    return false
  }
}

function validateNonEmptyString(value?: string) {
  if (!value) {
    return false
  }

  return value.length > 0
}

function validateTimeout(value?: string) {
  if (!value) {
    return true
  }

  try {
    const n = Number.parseInt(value)
    return n >= 0
  } catch {
    return false
  }
}

function validateCompressors(compressors?: string) {
  if (!compressors || compressors.length === 0) {
    return true
  }

  const list = compressors.indexOf(',') ? compressors.split(',') : [compressors]
  for (const compressor of list) {
    if (!SupportedCompressors.includes(compressor.toLowerCase())) {
      return false
    }
  }

  return true
}

function validatePositiveFloat(value?: string) {
  if (!value) {
    return false
  }

  try {
    const n = Number.parseFloat(value)
    return n >= 0
  } catch {
    return false
  }
}

function validateMaxStaleness(value?: string) {
  if (!value) {
    return true
  }

  if (value === '-1') {
    return true
  }

  return validatePositiveInteger(value)
}

function validateReadPreferenceMode(value?: string) {
  if (!value || value.length === 0) {
    return false
  }

  return ReadPreferenceMode.includes(value.toLowerCase())
}

function validateServerMonitoringMode(value?: string) {
  if (!value || value.length === 0) {
    return false
  }

  return ['auto', 'stream', 'poll'].includes(value.toLowerCase())
}

function validateZlibCompressionLevel(value?: string) {
  if (!value) {
    return false
  }

  try {
    const n = Number.parseInt(value)
    return n >= 0 && n <= 9
  } catch {
    return false
  }
}

export interface DetectAuthResult {
  authMethod: AuthMethod
  uri: string
}

export function detectAuthFromUri(uri: string): DetectAuthResult {
  const lower = uri.toLowerCase()
  if (lower.includes('authmechanism=mongodb-oidc')) {
    return { authMethod: 'oidc', uri }
  }
  if (lower.includes('authmechanism=mongodb-x509')) {
    return { authMethod: 'x509', uri }
  }
  if (lower.includes('authmechanism=mongodb-aws')) {
    return { authMethod: 'aws', uri }
  }
  if (lower.includes('authmechanism=gssapi')) {
    return { authMethod: 'gssapi', uri }
  }
  if (lower.includes('authmechanism=plain')) {
    return { authMethod: 'plain', uri }
  }
  return { authMethod: 'password', uri }
}

export function getUriHost(uri: string): string {
  if (!uri) {
    return ''
  }
  const result = parseUri(uri)
  if (!result.success || !result.data) {
    return ''
  }
  const first = result.data.nodelist[0]
  if (!first) {
    return ''
  }
  if (first.port != null) {
    return `${first.host}:${first.port}`
  }
  return first.host
}

export type SyncableAuthMechanism =
  | 'MONGODB-OIDC'
  | 'MONGODB-X509'
  | 'MONGODB-AWS'
  | 'GSSAPI'
  | 'PLAIN'

// The MongoDB connection-string spec (and Go driver's connstring.Parse) requires
// a "/" between the host list and the "?" that starts the query. Naive string
// joins like `${host}?${query}` produce "mongodb://host?..." which the driver
// rejects. Insert the slash whenever the host portion has no path segment.
export function ensurePathSlash(uri: string): string {
  const schemeMatch = uri.match(/^(mongodb(?:\+srv)?:\/\/)/)
  if (!schemeMatch) {
    return uri
  }
  const scheme = schemeMatch[1]!
  const rest = uri.substring(scheme.length)
  if (rest.indexOf('/') !== -1) {
    return uri
  }
  return `${scheme}${rest}/`
}

export function setAuthMechanism(uri: string, mechanism: SyncableAuthMechanism | null): string {
  const queryIdx = uri.indexOf('?')
  const base = queryIdx === -1 ? uri : uri.substring(0, queryIdx)
  const query = queryIdx === -1 ? '' : uri.substring(queryIdx + 1)

  const parts = query.length > 0 ? query.split('&') : []
  let hasAuthMechanism = false
  const kept: string[] = []
  for (const part of parts) {
    const lower = part.toLowerCase()
    if (lower.startsWith('authmechanism=')) {
      if (mechanism !== null && !hasAuthMechanism) {
        kept.push(`authMechanism=${mechanism}`)
        hasAuthMechanism = true
      }
      continue
    }
    if (lower.startsWith('authmechanismproperties=')) {
      continue
    }
    kept.push(part)
  }

  if (mechanism !== null && !hasAuthMechanism) {
    kept.push(`authMechanism=${mechanism}`)
  }

  return kept.length === 0 ? base : `${ensurePathSlash(base)}?${kept.join('&')}`
}

// Strip auth-related state (userinfo and auth/cert query params) so the field
// component for the newly-picked method starts from a clean URI. Without this
// switching from password → OIDC leaves stale "user:pass@" in the URI, and
// switching away from x509 leaves dangling tlsCertificateKey* options.
const AUTH_QUERY_PARAMS = [
  'authMechanism',
  'authMechanismProperties',
  'authSource',
  'tlsCertificateKeyFile',
  'tlsCertificateKeyFilePassword',
]

export function clearAuthState(uri: string): string {
  const schemeMatch = uri.match(/^(mongodb(?:\+srv)?:\/\/)/)
  const scheme = schemeMatch ? schemeMatch[1]! : 'mongodb://'
  const rest = schemeMatch ? uri.substring(scheme.length) : uri

  const queryIdx = rest.indexOf('?')
  const beforeQuery = queryIdx === -1 ? rest : rest.substring(0, queryIdx)
  const queryStr = queryIdx === -1 ? '' : rest.substring(queryIdx + 1)

  const atIdx = beforeQuery.lastIndexOf('@')
  const hostAndPath = atIdx === -1 ? beforeQuery : beforeQuery.substring(atIdx + 1)

  const stripLower = new Set(AUTH_QUERY_PARAMS.map((p) => p.toLowerCase()))
  const kept: string[] = []
  if (queryStr.length > 0) {
    for (const part of queryStr.split('&')) {
      const eq = part.indexOf('=')
      const key = eq === -1 ? part : part.substring(0, eq)
      if (stripLower.has(key.toLowerCase())) {
        continue
      }
      kept.push(part)
    }
  }

  const base = `${scheme}${hostAndPath}`
  return kept.length === 0 ? base : `${ensurePathSlash(base)}?${kept.join('&')}`
}

export type SignInBehaviour = 'openBrowser' | 'forceAccountPicker' | 'showUrl'

export function signInBehaviourFromConfig(cfg: OIDCConfig): SignInBehaviour {
  if (cfg.manualUrlMode === true) {
    return 'showUrl'
  }
  if (cfg.prompt === 'select_account') {
    return 'forceAccountPicker'
  }
  return 'openBrowser'
}

export function applySignInBehaviour(cfg: OIDCConfig, behaviour: SignInBehaviour): OIDCConfig {
  switch (behaviour) {
    case 'openBrowser':
      return { ...cfg, prompt: '', manualUrlMode: false }
    case 'forceAccountPicker':
      return { ...cfg, prompt: 'select_account', manualUrlMode: false }
    case 'showUrl':
      return { ...cfg, prompt: '', manualUrlMode: true }
  }
}

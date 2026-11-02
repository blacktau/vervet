const AUTH_METHODS = ['none', 'password', 'x509', 'oidc', 'aws', 'gssapi', 'plain'] as const

export type AuthMethod = (typeof AUTH_METHODS)[number]

/** Go sends the method as a plain string, and configs saved before it existed leave it empty. */
export function toAuthMethod(value: string): AuthMethod {
  return AUTH_METHODS.find((method) => method === value) ?? 'password'
}

export type OIDCPrompt = '' | 'login' | 'select_account' | 'consent'

export interface OIDCConfig {
  providerUrl: string
  clientId: string
  scopes?: string[]
  workloadIdentity: boolean
  prompt?: OIDCPrompt
  manualUrlMode?: boolean
}

export interface ConnectionConfig {
  uri: string
  authMethod: AuthMethod
  oidcConfig?: OIDCConfig
  refreshToken?: string
}

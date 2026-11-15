import type * as models from 'bindings/vervet/internal/models'

export type AuthMethod = 'none' | 'password' | 'x509' | 'oidc' | 'aws' | 'gssapi' | 'plain'

/**
 * Go sends the method as a plain string, and configs saved before it existed leave it empty.
 * An unrecognised value passes through, so that saving the server doesn't overwrite it.
 */
export function toAuthMethod(value: string): AuthMethod {
  return (value || 'password') as AuthMethod
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

/**
 * The bindings type authMethod as a TypeScript enum, but the wire value is the
 * same string, so the union converts directly.
 */
export function toBackendConfig(config: ConnectionConfig): models.ConnectionConfig {
  return { ...config, authMethod: config.authMethod as models.AuthMethod }
}

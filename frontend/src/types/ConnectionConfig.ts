export type AuthMethod = 'none' | 'password' | 'x509' | 'oidc' | 'aws' | 'gssapi' | 'plain'

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

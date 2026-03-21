export type AuthMethod = 'none' | 'password' | 'x509' | 'oidc' | 'aws'

export interface OIDCConfig {
  providerUrl: string
  clientId: string
  scopes?: string[]
  workloadIdentity: boolean
}

export interface ConnectionConfig {
  uri: string
  authMethod: AuthMethod
  oidcConfig?: OIDCConfig
  refreshToken?: string
}

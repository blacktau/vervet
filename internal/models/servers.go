package models

type RegisteredServer struct {
	ID        string `json:"id" yaml:"id"`
	Name      string `json:"name" yaml:"name"`
	ParentID  string `json:"parentID,omitempty" yaml:"parentID,omitempty"`
	Colour    string `json:"colour" yaml:"colour"`
	IsGroup   bool   `json:"isGroup" yaml:"isGroup"`
	IsCluster bool   `json:"isCluster" yaml:"isCluster"`
	IsSrv     bool   `json:"isSrv" yaml:"isSrv"`
}

type RegisteredServerConnection struct {
	URI string `json:"uri" yaml:"uri"`
	RegisteredServer
}

//type ConnectionType int

//type AuthenticationMode int
//
//const (
//	ConnectionTypeStandalone ConnectionType = iota
//	ConnectionTypeReplicaSet
//	ConnectionTypeShardedCluster
//	ConnectionTypeDNSSeedlist // mongo+srv
//)
//
//const (
//	AuthenticationModeNone AuthenticationMode = iota
//	AuthenticationModeSCRAMSHA1
//	AuthenticationModeSCRAMSHA256
//	AuthenticationModeX509
//	AuthenticationModeGSSAPI
//	AuthenticationModePLAIN
//	AuthenticationModeAWS
//	AuthenticationModeOIDC
//)

//type ServerInformation struct {
//	ID                            string             `json:"id" yaml:"id"`
//	Name                          string             `json:"name" yaml:"name"`
//	ConnectionType                ConnectionType     `json:"connectionType" yaml:"connectionType"`
//	//Host                          string             `json:"host" yaml:"host"`
//	//Port                          int                `json:"port" yaml:"port"`
//	//AuthMechanism                 AuthenticationMode `json:"authMechanism" yaml:"authMechanism"`
//	//AuthMechanismProperties       map[string]string  `json:"authMechanismProperties" yaml:"authMechanismProperties"`
//	//AuthSourceDB                  string             `json:"authSourceDB" yaml:"authSourceDB"`
//	//Username                      *string            `json:"username" yaml:"username"`
//	//Password                      *string            `json:"password" yaml:"password"`
//	//TLS                           bool               `json:"tls" yaml:"tls"`
//	//TLSCertificateKeyFile         *string            `json:"tlsCertificateKeyFile" yaml:"tlsCertificateKeyFile"`
//	//TLSCertificateKeyFilePassword *string            `json:"tlsCertificateKeyFilePassword" yaml:"tlsCertificateKeyFilePassword"`
//	//TLSCAFile                     *string            `json:"tlsCAFile" yaml:"tlsCAFile"`
//	//TLSAllowInvalidCertificates   bool               `json:"tlsAllowInvalidCertificates" yaml:"tlsAllowInvalidCertificates"`
//	//TLSAllowInvalidHostnames      bool               `json:"tlsAllowInvalidHostnames" yaml:"tlsAllowInvalidHostnames"`
//	//TLSInsecure                   bool               `json:"tlsInsecure" yaml:"tlsInsecure"`
//}
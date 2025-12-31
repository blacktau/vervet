package servers

import (
	"fmt"
	"sync"
	"vervet/internal/infrastructure"

	"github.com/wailsapp/wails/v2/pkg/logger"
	"gopkg.in/yaml.v3"
)

type Store struct {
	cfgStore infrastructure.Store
	mutex    sync.Mutex
	log      logger.Logger
}

type RegisteredServer struct {
	ID        string `json:"id" yaml:"id"`
	Name      string `json:"name" yaml:"name"`
	IsGroup   bool   `json:"isGroup" yaml:"isGroup"`
	ParentID  string `json:"parentID,omitempty" yaml:"parentID,omitempty"`
	Color     string `json:"color" yaml:"color"`
	IsCluster bool   `json:"isCluster" yaml:"isCluster"`
	IsSrv     bool   `json:"isSrv" yaml:"isSrv"`
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

type ServerStore interface {
	LoadServers() ([]RegisteredServer, error)
	SaveServers(servers []RegisteredServer) error
}

func NewServerStore(log logger.Logger) (ServerStore, error) {
	cfgStore, err := infrastructure.NewStore("connections.yaml", log)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %v", err)
	}

	return &Store{
		cfgStore: cfgStore,
		log:      log,
	}, nil
}

func (s *Store) LoadServers() ([]RegisteredServer, error) {
	b, err := s.cfgStore.Read()
	if err != nil {
		return nil, fmt.Errorf("error loading registered servers: %v", err)
	}

	registeredServers := make([]RegisteredServer, 0)

	if len(b) <= 0 {
		return registeredServers, nil
	}

	if err = yaml.Unmarshal(b, &registeredServers); err != nil {
		return nil, fmt.Errorf("error parsing registered servers: %v", err)
	}

	return registeredServers, nil
}

func (s *Store) SaveServers(registeredServers []RegisteredServer) error {
	b, err := yaml.Marshal(&registeredServers)
	if err != nil {
		return fmt.Errorf("error marshalling registered servers: %v", err)
	}

	if err = s.cfgStore.Save(b); err != nil {
		return fmt.Errorf("error saving registered servers: %v", err)
	}

	return nil
}

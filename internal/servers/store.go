package servers

import (
	"fmt"
	"log/slog"
	"sync"
	"vervet/internal/infrastructure"
	"vervet/internal/logging"

	"gopkg.in/yaml.v3"
)

type Store struct {
	cfgStore infrastructure.Store
	mutex    sync.Mutex
	log      *slog.Logger
}

type RegisteredServer struct {
	ID        string `json:"id" yaml:"id"`
	Name      string `json:"name" yaml:"name"`
	IsGroup   bool   `json:"isGroup" yaml:"isGroup"`
	ParentID  string `json:"parentID,omitempty" yaml:"parentID,omitempty"`
	Colour    string `json:"colour" yaml:"colour"`
	IsCluster bool   `json:"isCluster" yaml:"isCluster"`
	IsSrv     bool   `json:"isSrv" yaml:"isSrv"`
}

type RegisteredServerConnection struct {
	RegisteredServer
	URI string `json:"uri" yaml:"uri"`
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

func NewServerStore(log *slog.Logger) (ServerStore, error) {
	logger := log.With(slog.String(logging.SourceKey, "ServerStore"))
	cfgStore, err := infrastructure.NewStore("connections.yaml", logger)
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %v", err)
	}

	return &Store{
		cfgStore: cfgStore,
		log:      logger,
	}, nil
}

func (s *Store) LoadServers() ([]RegisteredServer, error) {
	b, err := s.cfgStore.Read()
	if err != nil {
		s.log.Error("error loading registered servers", slog.Any("error", err))
		return nil, fmt.Errorf("error loading registered servers: %v", err)
	}

	registeredServers := make([]RegisteredServer, 0)

	if len(b) <= 0 {
		return registeredServers, nil
	}

	if err = yaml.Unmarshal(b, &registeredServers); err != nil {
		s.log.Error("error parsing registered servers", slog.Any("error", err))
		return nil, fmt.Errorf("error parsing registered servers: %v", err)
	}

	return registeredServers, nil
}

func (s *Store) SaveServers(registeredServers []RegisteredServer) error {
	s.log.Debug("Saving Registered Servers")
	b, err := yaml.Marshal(&registeredServers)
	if err != nil {
		s.log.Error("error marshalling registered servers", slog.Any("error", err))
		return fmt.Errorf("error marshalling registered servers: %v", err)
	}

	if err = s.cfgStore.Save(b); err != nil {
		s.log.Error("error saving registered servers", slog.Any("error", err))
		return fmt.Errorf("error saving registered servers: %v", err)
	}

	return nil
}

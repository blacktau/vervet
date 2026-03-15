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

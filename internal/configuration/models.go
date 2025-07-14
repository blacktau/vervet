package configuration

type RegisteredServer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID int    `json:"parentId"`
	IsFolder bool   `json:"isFolder"`
}

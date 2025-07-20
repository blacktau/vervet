package configuration

type RegisteredServer struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID int    `json:"parentId"`
	IsGroup  bool   `json:"isGroup"`
}

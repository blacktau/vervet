package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
)

// dbMethodNames lists all intercepted db-level method names so the proxy
// can distinguish them from collection name access.
var dbMethodNames = map[string]bool{
	"getName":              true,
	"getCollection":        true,
	"runCommand":           true,
	"adminCommand":         true,
	"getCollectionNames":   true,
	"getCollectionInfos":   true,
	"createCollection":     true,
	"dropDatabase":         true,
	"stats":                true,
	"version":              true,
	"getSiblingDB":         true,
	"getMongo":             true,
	"aggregate":            true,
	"createUser":           true,
	"dropUser":             true,
	"getUser":              true,
	"getUsers":             true,
	"updateUser":           true,
	"changeUserPassword":   true,
	"grantRolesToUser":         true,
	"revokeRolesFromUser":      true,
	"dropAllUsers":             true,
	"createRole":               true,
	"dropRole":                 true,
	"getRole":                  true,
	"getRoles":                 true,
	"updateRole":               true,
	"grantPrivilegesToRole":    true,
	"revokePrivilegesFromRole": true,
	"grantRolesToRole":         true,
	"revokeRolesFromRole":      true,
	"dropAllRoles":             true,
}

// newDatabaseProxy creates a Goja Proxy object that intercepts property access.
// Accessing db.someCollection returns a collection proxy for "someCollection".
// Known db-level methods are intercepted and return Go-backed functions.
func newDatabaseProxy(ec *execContext) goja.Value {
	proxy := ec.rt.NewProxy(ec.rt.NewObject(), &goja.ProxyTrapConfig{
		Get: func(target *goja.Object, property string, receiver goja.Value) (value goja.Value) {
			switch property {
			case "getName":
				return ec.rt.ToValue(func() string { return ec.dbName })
			case "getCollection":
				return ec.rt.ToValue(func(name string) goja.Value {
					return newCollectionProxy(ec, name)
				})
			case "runCommand":
				return ec.rt.ToValue(dbRunCommand(ec, ec.dbName))
			case "adminCommand":
				return ec.rt.ToValue(dbRunCommand(ec, "admin"))
			case "getCollectionNames":
				return ec.rt.ToValue(dbGetCollectionNames(ec))
			case "getCollectionInfos":
				return ec.rt.ToValue(dbGetCollectionInfos(ec))
			case "createCollection":
				return ec.rt.ToValue(dbCreateCollection(ec))
			case "dropDatabase":
				return ec.rt.ToValue(dbDropDatabase(ec))
			case "stats":
				return ec.rt.ToValue(dbStats(ec))
			case "version":
				return ec.rt.ToValue(dbVersion(ec))
			case "getSiblingDB":
				return ec.rt.ToValue(dbGetSiblingDB(ec))
			case "getMongo":
				return ec.rt.ToValue(dbGetMongo(ec))
			case "aggregate":
				return ec.rt.ToValue(dbAggregate(ec))
			case "createUser":
				return ec.rt.ToValue(dbCreateUser(ec))
			case "dropUser":
				return ec.rt.ToValue(dbDropUser(ec))
			case "getUser":
				return ec.rt.ToValue(dbGetUser(ec))
			case "getUsers":
				return ec.rt.ToValue(dbGetUsers(ec))
			case "updateUser":
				return ec.rt.ToValue(dbUpdateUser(ec))
			case "changeUserPassword":
				return ec.rt.ToValue(dbChangeUserPassword(ec))
			case "grantRolesToUser":
				return ec.rt.ToValue(dbGrantRolesToUser(ec))
			case "revokeRolesFromUser":
				return ec.rt.ToValue(dbRevokeRolesFromUser(ec))
			case "dropAllUsers":
				return ec.rt.ToValue(dbDropAllUsers(ec))
			case "createRole":
				return ec.rt.ToValue(dbCreateRole(ec))
			case "dropRole":
				return ec.rt.ToValue(dbDropRole(ec))
			case "getRole":
				return ec.rt.ToValue(dbGetRole(ec))
			case "getRoles":
				return ec.rt.ToValue(dbGetRoles(ec))
			case "updateRole":
				return ec.rt.ToValue(dbUpdateRole(ec))
			case "grantPrivilegesToRole":
				return ec.rt.ToValue(dbGrantPrivilegesToRole(ec))
			case "revokePrivilegesFromRole":
				return ec.rt.ToValue(dbRevokePrivilegesFromRole(ec))
			case "grantRolesToRole":
				return ec.rt.ToValue(dbGrantRolesToRole(ec))
			case "revokeRolesFromRole":
				return ec.rt.ToValue(dbRevokeRolesFromRole(ec))
			case "dropAllRoles":
				return ec.rt.ToValue(dbDropAllRoles(ec))
			}
			return newCollectionProxy(ec, property)
		},
	})

	return ec.rt.ToValue(proxy)
}

func requireClient(ec *execContext) {
	if ec.client == nil {
		panic(ec.rt.NewGoError(fmt.Errorf("no MongoDB client available")))
	}
}

package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

// buildRoleCommand constructs a createRole command doc from a mongosh-style role spec.
func buildRoleCommand(roleDoc any) (bson.D, error) {
	m, ok := roleDoc.(map[string]any)
	if !ok {
		if d, ok := roleDoc.(bson.D); ok {
			m = map[string]any{}
			for _, e := range d {
				m[e.Key] = e.Value
			}
		} else if bm, ok := roleDoc.(bson.M); ok {
			m = map[string]any(bm)
		} else {
			return nil, fmt.Errorf("createRole requires a role document")
		}
	}
	name, ok := m["role"].(string)
	if !ok {
		return nil, fmt.Errorf("createRole requires a 'role' field")
	}
	cmd := bson.D{{Key: "createRole", Value: name}}
	for k, v := range m {
		if k == "role" {
			continue
		}
		cmd = append(cmd, bson.E{Key: k, Value: v})
	}
	return cmd, nil
}

func buildUpdateRoleCommand(name string, updateDoc any) (bson.D, error) {
	cmd := bson.D{{Key: "updateRole", Value: name}}
	if updateDoc == nil {
		return cmd, nil
	}
	m, ok := updateDoc.(map[string]any)
	if !ok {
		if d, ok := updateDoc.(bson.D); ok {
			m = map[string]any{}
			for _, e := range d {
				m[e.Key] = e.Value
			}
		} else if bm, ok := updateDoc.(bson.M); ok {
			m = map[string]any(bm)
		} else {
			return nil, fmt.Errorf("updateRole requires an update document")
		}
	}
	for k, v := range m {
		cmd = append(cmd, bson.E{Key: k, Value: v})
	}
	return cmd, nil
}

func dbCreateRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("createRole requires a role document")))
		}
		cmd, err := buildRoleCommand(exportValue(call.Arguments[0]))
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("createRole: %w", err)))
		}
		return ec.rt.ToValue(runDBCommand(ec, "createRole", cmd))
	}
}

func dbDropRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("dropRole requires a role name")))
		}
		name := call.Arguments[0].String()
		cmd := bson.D{{Key: "dropRole", Value: name}}
		return ec.rt.ToValue(runDBCommand(ec, "dropRole", cmd))
	}
}

func dbGetRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("getRole requires a role name")))
		}
		name := call.Arguments[0].String()
		cmd := bson.D{{Key: "rolesInfo", Value: name}}
		result := runDBCommand(ec, "getRole", cmd)
		roles, _ := result["roles"].(bson.A)
		if len(roles) == 0 {
			return goja.Null()
		}
		return ec.rt.ToValue(roles[0])
	}
}

func dbGetRoles(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "rolesInfo", Value: 1}}
		result := runDBCommand(ec, "getRoles", cmd)
		roles, _ := result["roles"].(bson.A)
		out := make([]any, len(roles))
		for i, r := range roles {
			out[i] = r
		}
		return ec.rt.ToValue(out)
	}
}

func dbUpdateRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("updateRole requires role name and update document")))
		}
		name := call.Arguments[0].String()
		cmd, err := buildUpdateRoleCommand(name, exportValue(call.Arguments[1]))
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("updateRole: %w", err)))
		}
		return ec.rt.ToValue(runDBCommand(ec, "updateRole", cmd))
	}
}

func dbGrantPrivilegesToRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("grantPrivilegesToRole requires role name and privileges")))
		}
		name := call.Arguments[0].String()
		privs := convertToBson(exportValue(call.Arguments[1]))
		cmd := bson.D{{Key: "grantPrivilegesToRole", Value: name}, {Key: "privileges", Value: privs}}
		return ec.rt.ToValue(runDBCommand(ec, "grantPrivilegesToRole", cmd))
	}
}

func dbRevokePrivilegesFromRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("revokePrivilegesFromRole requires role name and privileges")))
		}
		name := call.Arguments[0].String()
		privs := convertToBson(exportValue(call.Arguments[1]))
		cmd := bson.D{{Key: "revokePrivilegesFromRole", Value: name}, {Key: "privileges", Value: privs}}
		return ec.rt.ToValue(runDBCommand(ec, "revokePrivilegesFromRole", cmd))
	}
}

func dbGrantRolesToRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("grantRolesToRole requires role name and roles")))
		}
		name := call.Arguments[0].String()
		roles := convertToBson(exportValue(call.Arguments[1]))
		cmd := bson.D{{Key: "grantRolesToRole", Value: name}, {Key: "roles", Value: roles}}
		return ec.rt.ToValue(runDBCommand(ec, "grantRolesToRole", cmd))
	}
}

func dbRevokeRolesFromRole(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("revokeRolesFromRole requires role name and roles")))
		}
		name := call.Arguments[0].String()
		roles := convertToBson(exportValue(call.Arguments[1]))
		cmd := bson.D{{Key: "revokeRolesFromRole", Value: name}, {Key: "roles", Value: roles}}
		return ec.rt.ToValue(runDBCommand(ec, "revokeRolesFromRole", cmd))
	}
}

func dbDropAllRoles(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "dropAllRolesFromDatabase", Value: 1}}
		return ec.rt.ToValue(runDBCommand(ec, "dropAllRoles", cmd))
	}
}

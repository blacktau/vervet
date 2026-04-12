package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/bson"
)

// buildUserCommand constructs a createUser command doc from a mongosh-style user spec.
// The user doc is spread into the command with "createUser" replacing the "user" field as the command value.
func buildUserCommand(userDoc any) (bson.D, error) {
	m, ok := userDoc.(map[string]any)
	if !ok {
		if d, ok := userDoc.(bson.D); ok {
			m = map[string]any{}
			for _, e := range d {
				m[e.Key] = e.Value
			}
		} else if bm, ok := userDoc.(bson.M); ok {
			m = map[string]any(bm)
		} else {
			return nil, fmt.Errorf("createUser requires a user document")
		}
	}
	name, ok := m["user"].(string)
	if !ok {
		return nil, fmt.Errorf("createUser requires a 'user' field")
	}
	cmd := bson.D{{Key: "createUser", Value: name}}
	for k, v := range m {
		if k == "user" {
			continue
		}
		cmd = append(cmd, bson.E{Key: k, Value: v})
	}
	return cmd, nil
}

// buildUpdateUserCommand constructs an updateUser command doc.
func buildUpdateUserCommand(name string, updateDoc any) (bson.D, error) {
	cmd := bson.D{{Key: "updateUser", Value: name}}
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
			return nil, fmt.Errorf("updateUser requires an update document")
		}
	}
	for k, v := range m {
		cmd = append(cmd, bson.E{Key: k, Value: v})
	}
	return cmd, nil
}

func runDBCommand(ec *execContext, methodName string, cmd bson.D) bson.M {
	var result bson.M
	err := ec.client.Database(ec.dbName).RunCommand(ec.ctx, cmd).Decode(&result)
	if err != nil {
		panic(ec.rt.NewGoError(fmt.Errorf("%s: %w", methodName, err)))
	}
	return result
}

func dbCreateUser(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("createUser requires a user document")))
		}
		cmd, err := buildUserCommand(exportValue(call.Arguments[0]))
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("createUser: %w", err)))
		}
		return ec.rt.ToValue(runDBCommand(ec, "createUser", cmd))
	}
}

func dbDropUser(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("dropUser requires a username")))
		}
		name := call.Arguments[0].String()
		cmd := bson.D{{Key: "dropUser", Value: name}}
		return ec.rt.ToValue(runDBCommand(ec, "dropUser", cmd))
	}
}

func dbGetUser(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) == 0 {
			panic(ec.rt.NewGoError(fmt.Errorf("getUser requires a username")))
		}
		name := call.Arguments[0].String()
		cmd := bson.D{{Key: "usersInfo", Value: name}}
		result := runDBCommand(ec, "getUser", cmd)
		users, _ := result["users"].(bson.A)
		if len(users) == 0 {
			return goja.Null()
		}
		return ec.rt.ToValue(users[0])
	}
}

func dbGetUsers(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "usersInfo", Value: 1}}
		result := runDBCommand(ec, "getUsers", cmd)
		users, _ := result["users"].(bson.A)
		out := make([]any, len(users))
		for i, u := range users {
			out[i] = u
		}
		return ec.rt.ToValue(out)
	}
}

func dbUpdateUser(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("updateUser requires username and update document")))
		}
		name := call.Arguments[0].String()
		cmd, err := buildUpdateUserCommand(name, exportValue(call.Arguments[1]))
		if err != nil {
			panic(ec.rt.NewGoError(fmt.Errorf("updateUser: %w", err)))
		}
		return ec.rt.ToValue(runDBCommand(ec, "updateUser", cmd))
	}
}

func dbChangeUserPassword(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("changeUserPassword requires username and password")))
		}
		name := call.Arguments[0].String()
		pwd := call.Arguments[1].String()
		cmd := bson.D{{Key: "updateUser", Value: name}, {Key: "pwd", Value: pwd}}
		return ec.rt.ToValue(runDBCommand(ec, "changeUserPassword", cmd))
	}
}

func dbGrantRolesToUser(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("grantRolesToUser requires username and roles")))
		}
		name := call.Arguments[0].String()
		roles := convertToBson(exportValue(call.Arguments[1]))
		cmd := bson.D{{Key: "grantRolesToUser", Value: name}, {Key: "roles", Value: roles}}
		return ec.rt.ToValue(runDBCommand(ec, "grantRolesToUser", cmd))
	}
}

func dbRevokeRolesFromUser(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		if len(call.Arguments) < 2 {
			panic(ec.rt.NewGoError(fmt.Errorf("revokeRolesFromUser requires username and roles")))
		}
		name := call.Arguments[0].String()
		roles := convertToBson(exportValue(call.Arguments[1]))
		cmd := bson.D{{Key: "revokeRolesFromUser", Value: name}, {Key: "roles", Value: roles}}
		return ec.rt.ToValue(runDBCommand(ec, "revokeRolesFromUser", cmd))
	}
}

func dbDropAllUsers(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "dropAllUsersFromDatabase", Value: 1}}
		return ec.rt.ToValue(runDBCommand(ec, "dropAllUsers", cmd))
	}
}

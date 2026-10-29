package queryengine

import (
	"fmt"

	"github.com/dop251/goja"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// buildUserCommand constructs a createUser command doc from a mongosh-style user spec.
// The user doc is spread into the command with "createUser" replacing the "user" field as the command value.
func buildUserCommand(userDoc any) (bson.D, error) {
	return buildNamedCommand("createUser", "user", userDoc)
}

// buildUpdateUserCommand constructs an updateUser command doc.
func buildUpdateUserCommand(name string, updateDoc any) (bson.D, error) {
	return buildUpdateCommand("updateUser", name, updateDoc)
}

// buildNamedCommand builds a command whose value is taken from the spec's
// nameField (e.g. createUser from "user"), followed by the spec's remaining
// fields in their original order.
func buildNamedCommand(cmdName, nameField string, spec any) (bson.D, error) {
	doc, ok := asBsonDoc(spec)
	if !ok {
		return nil, fmt.Errorf("%s requires a %s document", cmdName, nameField)
	}
	cmd := bson.D{{Key: cmdName}}
	for _, e := range doc {
		if e.Key == nameField {
			cmd[0].Value = e.Value
			continue
		}
		cmd = append(cmd, e)
	}
	if _, ok := cmd[0].Value.(string); !ok {
		return nil, fmt.Errorf("%s requires a '%s' field", cmdName, nameField)
	}
	return cmd, nil
}

// buildUpdateCommand builds { cmdName: name, ...updateDoc } keeping the
// update document's field order.
func buildUpdateCommand(cmdName, name string, updateDoc any) (bson.D, error) {
	cmd := bson.D{{Key: cmdName, Value: name}}
	if updateDoc == nil {
		return cmd, nil
	}
	doc, ok := asBsonDoc(updateDoc)
	if !ok {
		return nil, fmt.Errorf("%s requires an update document", cmdName)
	}
	return append(cmd, doc...), nil
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
		return toJSValue(ec.rt, runDBCommand(ec, "createUser", cmd))
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
		return toJSValue(ec.rt, runDBCommand(ec, "dropUser", cmd))
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
		return toJSValue(ec.rt, users[0])
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
		return toJSValue(ec.rt, out)
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
		return toJSValue(ec.rt, runDBCommand(ec, "updateUser", cmd))
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
		return toJSValue(ec.rt, runDBCommand(ec, "changeUserPassword", cmd))
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
		return toJSValue(ec.rt, runDBCommand(ec, "grantRolesToUser", cmd))
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
		return toJSValue(ec.rt, runDBCommand(ec, "revokeRolesFromUser", cmd))
	}
}

func dbDropAllUsers(ec *execContext) func(goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		requireClient(ec)
		cmd := bson.D{{Key: "dropAllUsersFromDatabase", Value: 1}}
		return toJSValue(ec.rt, runDBCommand(ec, "dropAllUsers", cmd))
	}
}

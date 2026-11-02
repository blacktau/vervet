package queryengine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dop251/goja"

	"vervet/internal/models"
)

// scriptOutput collects everything a script writes with print, printjson and
// console.*, in order, one entry per call. console.warn and console.error keep
// their level so the Messages pane can show them as warnings and errors.
type scriptOutput struct {
	lines []models.OutputLine
}

func (o *scriptOutput) add(level, text string) {
	o.lines = append(o.lines, models.OutputLine{Level: level, Text: text})
}

// consoleLevels maps each console method to the level its output is shown at.
var consoleLevels = map[string]string{
	"log":   models.OutputInfo,
	"info":  models.OutputInfo,
	"debug": models.OutputInfo,
	"trace": models.OutputInfo,
	"warn":  models.OutputWarning,
	"error": models.OutputError,
}

// registerOutput installs print, printjson and console in the runtime.
func registerOutput(rt *goja.Runtime, out *scriptOutput) error {
	// printAt joins its arguments with a space onto one line, as mongosh's
	// print does, and records it at level.
	printAt := func(level string) func(goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			parts := make([]string, len(call.Arguments))
			for i, arg := range call.Arguments {
				parts[i] = printArg(rt, arg)
			}
			out.add(level, strings.Join(parts, " "))
			return goja.Undefined()
		}
	}
	printFn := printAt(models.OutputInfo)

	if err := rt.Set("print", printFn); err != nil {
		return fmt.Errorf("failed to set print function: %w", err)
	}

	// printjson expands every non-empty object and array, one entry per line,
	// however short it is — mongosh does the same.
	if err := rt.Set("printjson", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, len(call.Arguments))
		for i, arg := range call.Arguments {
			parts[i] = printArgExpanded(rt, arg)
		}
		out.add(models.OutputInfo, strings.Join(parts, " "))
		return goja.Undefined()
	}); err != nil {
		return fmt.Errorf("failed to set printjson function: %w", err)
	}

	console := rt.NewObject()
	for method, level := range consoleLevels {
		if err := console.Set(method, printAt(level)); err != nil {
			return fmt.Errorf("failed to set console.%s: %w", method, err)
		}
	}
	if err := rt.Set("console", console); err != nil {
		return fmt.Errorf("failed to set console global: %w", err)
	}

	return nil
}

// printArg renders one print argument: strings are written as-is (print("a")
// gives a, not 'a'), everything else is inspected.
func printArg(rt *goja.Runtime, v goja.Value) string {
	return printArgWith(rt, v, false)
}

// printArgExpanded is printArg with containers always broken over lines.
func printArgExpanded(rt *goja.Runtime, v goja.Value) string {
	return printArgWith(rt, v, true)
}

func printArgWith(rt *goja.Runtime, v goja.Value, expand bool) string {
	if v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if _, ok := v.Export().(string); ok {
			return v.String()
		}
	}
	return inspectValue(rt, v, 0, expand)
}

// maxInlineWidth is where inspect switches an object or array to one entry per
// line, mirroring how mongosh wraps wide values.
const maxInlineWidth = 72

// inspect renders a value the way the mongosh shell displays it: unquoted keys,
// single-quoted strings, and BSON values as their own string form.
func inspect(rt *goja.Runtime, v goja.Value, depth int) string {
	return inspectValue(rt, v, depth, false)
}

// inspectValue renders v; expand forces every non-empty object and array onto
// multiple lines regardless of width.
func inspectValue(rt *goja.Runtime, v goja.Value, depth int, expand bool) string {
	if v == nil || goja.IsUndefined(v) {
		return "undefined"
	}
	if goja.IsNull(v) {
		return "null"
	}

	obj, isObj := v.(*goja.Object)
	if !isObj {
		return inspectPrimitive(v)
	}

	switch obj.ClassName() {
	case "Date":
		return "ISODate(\"" + callMethod(rt, obj, "toISOString") + "\")"
	case "RegExp", "Function", "Error":
		return callToString(rt, obj)
	}

	// BSON wrappers (ObjectId, Long, Decimal128, ...) print as their value.
	if bv := obj.Get("__bsonValue"); bv != nil && !goja.IsUndefined(bv) {
		return callToString(rt, obj)
	}

	if depth > 8 {
		return "[Object]"
	}

	if obj.ClassName() == "Array" {
		return inspectArray(rt, obj, depth, expand)
	}
	return inspectObject(rt, obj, depth, expand)
}

func inspectPrimitive(v goja.Value) string {
	switch val := v.Export().(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "\\'") + "'"
	case bool:
		return strconv.FormatBool(val)
	default:
		return v.String()
	}
}

func inspectArray(rt *goja.Runtime, obj *goja.Object, depth int, expand bool) string {
	length := int(obj.Get("length").ToInteger())
	if length == 0 {
		return "[]"
	}
	items := make([]string, length)
	for i := 0; i < length; i++ {
		items[i] = inspectValue(rt, obj.Get(strconv.Itoa(i)), depth+1, expand)
	}
	return wrap("[", items, "]", depth, expand)
}

func inspectObject(rt *goja.Runtime, obj *goja.Object, depth int, expand bool) string {
	keys := obj.Keys()
	items := make([]string, 0, len(keys))
	for _, key := range keys {
		val := obj.Get(key)
		if val != nil && !goja.IsUndefined(val) {
			if _, isFunc := goja.AssertFunction(val); isFunc {
				continue
			}
		}
		items = append(items, inspectKey(key)+": "+inspectValue(rt, val, depth+1, expand))
	}
	if len(items) == 0 {
		return "{}"
	}
	return wrap("{", items, "}", depth, expand)
}

// inspectKey quotes only keys that are not plain identifiers.
func inspectKey(key string) string {
	for i, c := range key {
		isLetter := c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isLetter && !(isDigit && i > 0) {
			return "'" + key + "'"
		}
	}
	if key == "" {
		return "''"
	}
	return key
}

// wrap lays entries out on one line when they fit, otherwise one per line.
func wrap(open string, items []string, closing string, depth int, expand bool) string {
	inline := open + " " + strings.Join(items, ", ") + " " + closing
	if !expand && len(inline)+depth*2 <= maxInlineWidth && !strings.Contains(inline, "\n") {
		return inline
	}
	pad := strings.Repeat("  ", depth+1)
	return open + "\n" + pad + strings.Join(items, ",\n"+pad) + "\n" + strings.Repeat("  ", depth) + closing
}

// callToString invokes the value's own toString, falling back to Goja's.
func callToString(rt *goja.Runtime, obj *goja.Object) string {
	return callMethod(rt, obj, "toString")
}

// callMethod calls a no-argument string method on obj, falling back to the
// value's default string form when it is missing or throws.
func callMethod(rt *goja.Runtime, obj *goja.Object, name string) string {
	if fn, ok := goja.AssertFunction(obj.Get(name)); ok {
		if res, err := fn(obj); err == nil {
			return res.String()
		}
	}
	return obj.String()
}

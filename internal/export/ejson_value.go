package export

import (
	"bytes"
	"encoding/json"
	"strings"
)

type ejsonWrapper struct {
	isPayload func(json.RawMessage) bool
	// companion is a second key the wrapper may carry, as in {$regex, $options}.
	companion string
}

// ejsonWrappers matches bsonValue.ts in the frontend, so a result the results
// tree shows as a single value is exported as one.
var ejsonWrappers = map[string]ejsonWrapper{
	"$oid":               {isPayload: isJSONString},
	"$date":              {isPayload: isEJSONDatePayload},
	"$numberDecimal":     {isPayload: isJSONString},
	"$numberLong":        {isPayload: isJSONString},
	"$numberInt":         {isPayload: isJSONString},
	"$numberDouble":      {isPayload: isJSONString},
	"$binary":            {isPayload: objectWithString("base64")},
	"$regex":             {isPayload: isJSONString, companion: "$options"},
	"$regularExpression": {isPayload: objectWithString("pattern")},
	"$timestamp":         {isPayload: objectWithKeys("t", "i")},
	"$minKey":            {isPayload: isJSONLiteral("1")},
	"$maxKey":            {isPayload: isJSONLiteral("1")},
	"$code":              {isPayload: isJSONString, companion: "$scope"},
	"$symbol":            {isPayload: isJSONString},
	"$undefined":         {isPayload: isJSONLiteral("true")},
	"$uuid":              {isPayload: isJSONString},
	"$dbPointer":         {isPayload: isEJSONDBPointerPayload},
}

// parseEJSONValueFields returns the fields of raw when it is exactly an EJSON
// wrapper. Real documents almost always start with a key such as _id, so this
// reads only the first key of those.
func parseEJSONValueFields(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	if !firstKeyHasDollar(raw) {
		return nil, false
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, false
	}

	for key, wrapper := range ejsonWrappers {
		payload, found := fields[key]
		if !found {
			continue
		}
		for other := range fields {
			if other != key && other != wrapper.companion {
				return nil, false
			}
		}
		return fields, wrapper.isPayload(payload)
	}

	return nil, false
}

func firstKeyHasDollar(raw json.RawMessage) bool {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if _, err := decoder.Token(); err != nil {
		return false
	}
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	key, isString := token.(string)
	return isString && strings.HasPrefix(key, "$")
}

func isJSONString(raw json.RawMessage) bool {
	var s string
	return len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, &s) == nil
}

func isJSONLiteral(want string) func(json.RawMessage) bool {
	return func(raw json.RawMessage) bool {
		return string(raw) == want
	}
}

func jsonObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var fields map[string]json.RawMessage
	if len(raw) == 0 || raw[0] != '{' || json.Unmarshal(raw, &fields) != nil {
		return nil, false
	}
	return fields, true
}

func objectWithString(key string) func(json.RawMessage) bool {
	return func(raw json.RawMessage) bool {
		fields, isObject := jsonObject(raw)
		return isObject && isJSONString(fields[key])
	}
}

func objectWithKeys(keys ...string) func(json.RawMessage) bool {
	return func(raw json.RawMessage) bool {
		fields, isObject := jsonObject(raw)
		if !isObject {
			return false
		}
		for _, key := range keys {
			if _, found := fields[key]; !found {
				return false
			}
		}
		return true
	}
}

func isEJSONDatePayload(raw json.RawMessage) bool {
	if isJSONString(raw) || objectWithString("$numberLong")(raw) {
		return true
	}
	var number float64
	return len(raw) > 0 && raw[0] != 'n' && json.Unmarshal(raw, &number) == nil
}

func isEJSONDBPointerPayload(raw json.RawMessage) bool {
	fields, isObject := jsonObject(raw)
	if !isObject {
		return false
	}
	_, idIsObject := jsonObject(fields["$id"])
	return idIsObject && isJSONString(fields["$ref"])
}

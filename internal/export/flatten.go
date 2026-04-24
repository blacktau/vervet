package export

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

type flatPair struct {
	path  string
	value string
}

// flattenDoc converts a BSON document into a flat list of dot-path/value pairs.
// Nested objects are recursed with dot-joined paths. Arrays and BSON special types
// are serialised as canonical Extended JSON strings. Keys at each level are sorted
// alphabetically for deterministic output.
func flattenDoc(doc bson.M) ([]flatPair, error) {
	return walkDoc(doc, "")
}

func walkDoc(doc bson.M, prefix string) ([]flatPair, error) {
	keys := sortedKeys(doc)
	var pairs []flatPair
	for _, k := range keys {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		nested, err := walkValue(path, doc[k])
		if err != nil {
			return nil, err
		}
		pairs = append(pairs, nested...)
	}
	return pairs, nil
}

func walkValue(path string, v any) ([]flatPair, error) {
	switch val := v.(type) {
	case nil:
		return []flatPair{{path: path, value: ""}}, nil

	case string:
		return []flatPair{{path: path, value: val}}, nil

	case bool:
		return []flatPair{{path: path, value: strconv.FormatBool(val)}}, nil

	case int:
		return []flatPair{{path: path, value: strconv.FormatInt(int64(val), 10)}}, nil

	case int32:
		return []flatPair{{path: path, value: strconv.FormatInt(int64(val), 10)}}, nil

	case int64:
		return []flatPair{{path: path, value: strconv.FormatInt(val, 10)}}, nil

	case float32:
		return []flatPair{{path: path, value: strconv.FormatFloat(float64(val), 'f', -1, 64)}}, nil

	case float64:
		return []flatPair{{path: path, value: strconv.FormatFloat(val, 'f', -1, 64)}}, nil

	case bson.M:
		return walkDoc(val, path)

	case bson.A:
		s, err := marshalAsEJSONString(val)
		if err != nil {
			return nil, fmt.Errorf("path %q: %w", path, err)
		}
		return []flatPair{{path: path, value: s}}, nil

	default:
		// BSON special types (ObjectID, DateTime, Decimal128, Binary, etc.)
		s, err := marshalAsEJSONString(val)
		if err != nil {
			return nil, fmt.Errorf("path %q: %w", path, err)
		}
		return []flatPair{{path: path, value: s}}, nil
	}
}

// marshalAsEJSONString serialises an arbitrary BSON-compatible value as its
// canonical Extended JSON text representation (without surrounding braces).
func marshalAsEJSONString(v any) (string, error) {
	data, err := bson.MarshalExtJSON(bson.M{"v": v}, false, false)
	if err != nil {
		return "", fmt.Errorf("marshal ext-json: %w", err)
	}
	s := string(data)
	// data looks like `{"v":<value>}`. Strip the wrapper.
	const prefix = `{"v":`
	if !strings.HasPrefix(s, prefix) || !strings.HasSuffix(s, "}") {
		return s, nil
	}
	return s[len(prefix) : len(s)-1], nil
}

func sortedKeys(m bson.M) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortStrings sorts a slice of strings in-place alphabetically.
func sortStrings(s []string) {
	sort.Strings(s)
}

package queryengine

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// decodeOrderedJSON decodes a JSON document into the ordered tree the script
// path consumes: objects become bson.D and arrays []any. bson.D's own
// UnmarshalJSON is not used because it decodes an empty array as nil, which
// would turn `tags: []` into `tags: null`.
func decodeOrderedJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	return decodeJSONValue(dec)
}

func decodeJSONValue(dec *json.Decoder) (any, error) {
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return t, nil
	}
	switch delim {
	case '{':
		doc := bson.D{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := kt.(string)
			if !ok {
				return nil, fmt.Errorf("invalid JSON object key %v", kt)
			}
			val, err := decodeJSONValue(dec)
			if err != nil {
				return nil, err
			}
			doc = append(doc, bson.E{Key: key, Value: val})
		}
		_, err = dec.Token() // closing '}'
		return doc, err
	case '[':
		arr := []any{}
		for dec.More() {
			val, err := decodeJSONValue(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		_, err = dec.Token() // closing ']'
		return arr, err
	default:
		return nil, fmt.Errorf("unexpected JSON delimiter %v", delim)
	}
}

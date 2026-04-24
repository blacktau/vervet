package export

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

// serializeJSONImpl renders docs as a pretty-printed Extended JSON array.
// Uses bson.MarshalExtJSON per doc to preserve BSON types, then re-parses
// via encoding/json to pretty-print the array container.
func serializeJSONImpl(docs []bson.M) ([]byte, error) {
	if docs == nil || len(docs) == 0 {
		return []byte("[]"), nil
	}

	raws := make([]json.RawMessage, len(docs))
	for i, d := range docs {
		data, err := bson.MarshalExtJSON(d, false, false)
		if err != nil {
			return nil, fmt.Errorf("marshal doc %d: %w", i, err)
		}
		raws[i] = data
	}

	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, r := range raws {
		if i > 0 {
			buf.WriteString(",\n  ")
		} else {
			buf.WriteString("\n  ")
		}
		// pretty-print the doc with 2-space indent, nested under the array.
		var parsed any
		if err := json.Unmarshal(r, &parsed); err != nil {
			return nil, fmt.Errorf("parse ext-json doc %d: %w", i, err)
		}
		pretty, err := json.MarshalIndent(parsed, "  ", "  ")
		if err != nil {
			return nil, fmt.Errorf("pretty-print doc %d: %w", i, err)
		}
		buf.Write(pretty)
	}
	buf.WriteString("\n]")
	return buf.Bytes(), nil
}

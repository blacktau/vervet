package export

import (
	"bytes"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func serializeNDJSON(docs []bson.M) ([]byte, error) {
	if len(docs) == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	for i, d := range docs {
		data, err := bson.MarshalExtJSON(d, false, false)
		if err != nil {
			return nil, fmt.Errorf("marshal doc %d: %w", i, err)
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

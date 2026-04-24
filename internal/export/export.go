package export

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func Serialize(docs []bson.M, opts Options) ([]byte, error) {
	switch opts.Format {
	case FormatJSON:
		return serializeJSON(docs)
	case FormatNDJSON:
		return nil, fmt.Errorf("ndjson not yet implemented")
	case FormatCSV:
		return nil, fmt.Errorf("csv not yet implemented")
	default:
		return nil, fmt.Errorf("unknown format %q", opts.Format)
	}
}

// serializeJSON is a temporary stub so the package compiles.
// The real implementation lands in Task 2.
func serializeJSON(docs []bson.M) ([]byte, error) {
	if docs == nil {
		docs = []bson.M{}
	}
	if len(docs) == 0 {
		return []byte("[]"), nil
	}
	return nil, fmt.Errorf("json not yet implemented")
}

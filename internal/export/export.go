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
		return serializeNDJSON(docs)
	case FormatCSV:
		return nil, fmt.Errorf("csv not yet implemented")
	default:
		return nil, fmt.Errorf("unknown format %q", opts.Format)
	}
}

func serializeJSON(docs []bson.M) ([]byte, error) {
	return serializeJSONImpl(docs)
}

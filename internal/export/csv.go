package export

import (
	"bytes"
	"encoding/csv"

	"go.mongodb.org/mongo-driver/bson"
)

func serializeCSV(docs []bson.M, columns []string, opts CSVOptions) ([]byte, error) {
	// Flatten each doc into path→value pairs.
	rows := make([]map[string]string, len(docs))
	for i, d := range docs {
		pairs, err := flattenDoc(d)
		if err != nil {
			return nil, err
		}
		m := make(map[string]string, len(pairs))
		for _, p := range pairs {
			m[p.path] = p.value
		}
		rows[i] = m
	}

	// Derive header: explicit columns win; otherwise sorted union of keys.
	var header []string
	if len(columns) > 0 {
		header = columns
	} else {
		seen := map[string]struct{}{}
		for _, r := range rows {
			for k := range r {
				seen[k] = struct{}{}
			}
		}
		header = make([]string, 0, len(seen))
		for k := range seen {
			header = append(header, k)
		}
		sortedKeys := make([]string, len(header))
		copy(sortedKeys, header)
		sortStrings(sortedKeys)
		header = sortedKeys
	}

	// Pick separator (0 → comma).
	sep := opts.Separator
	if sep == 0 {
		sep = ','
	}

	var buf bytes.Buffer
	if opts.UTF8BOM {
		buf.WriteString("\xEF\xBB\xBF")
	}

	w := csv.NewWriter(&buf)
	w.Comma = sep

	if opts.IncludeHeader {
		if err := w.Write(header); err != nil {
			return nil, err
		}
	}
	for _, r := range rows {
		record := make([]string, len(header))
		for i, col := range header {
			record[i] = r[col] // missing → zero value ""
		}
		if err := w.Write(record); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

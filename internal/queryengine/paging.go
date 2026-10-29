package queryengine

import (
	"context"
	"fmt"

	"vervet/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// encodePageDoc renders a cursor's filter/projection/sort/hint document as
// canonical Extended JSON for a PageContext. Nil and non-documents encode as
// "" (absent), matching how dispatch treats them (toBsonDoc yields an empty
// document), so later pages run the same query as the first.
func encodePageDoc(v any) (string, error) {
	doc, ok := asBsonDoc(v)
	if !ok {
		return "", nil
	}
	data, err := bson.MarshalExtJSON(doc, true, false)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// decodePageDoc is the inverse of encodePageDoc. "" decodes as nil (absent),
// so an omitted projection or sort stays omitted.
func decodePageDoc(s string) (any, error) {
	doc, err := unmarshalPageJSON(s)
	if doc == nil {
		return nil, err
	}
	return doc, err
}

// encodePageValue renders any cursor value (the hint may be an index name, a
// key document or something else) as canonical Extended JSON, wrapped as
// {"v": value}. Nil encodes as "" (absent).
func encodePageValue(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	data, err := bson.MarshalExtJSON(bson.D{{Key: "v", Value: convertToBson(v)}}, true, false)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// decodePageValue is the inverse of encodePageValue.
func decodePageValue(s string) (any, error) {
	doc, err := unmarshalPageJSON(s)
	if doc == nil {
		return nil, err
	}
	return docField(doc, "v"), err
}

// unmarshalPageJSON decodes a PageContext field; "" yields nil.
func unmarshalPageJSON(s string) (bson.D, error) {
	if s == "" {
		return nil, nil
	}
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(s), true, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// buildPageContext snapshots this cursor's find parameters so the frontend
// can fetch later pages. Returns nil for findOne (not pageable). A cursor
// option that should apply to every page must be copied here and applied in
// FetchPage.
func (c *lazyCursor) buildPageContext() (*models.PageContext, error) {
	if c.isFindOne {
		return nil, nil
	}
	pc := &models.PageContext{
		Collection: c.collection,
		UserLimit:  c.limit,
		UserSkip:   c.skip,
		MaxTimeMS:  c.maxTimeMS,
		BatchSize:  c.batchSize,
		Comment:    c.comment,
	}
	var err error
	if pc.Filter, err = encodePageDoc(c.filter); err != nil {
		return nil, fmt.Errorf("paging filter: %w", err)
	}
	if pc.Projection, err = encodePageDoc(c.projection); err != nil {
		return nil, fmt.Errorf("paging projection: %w", err)
	}
	if pc.Sort, err = encodePageDoc(c.sort); err != nil {
		return nil, fmt.Errorf("paging sort: %w", err)
	}
	if pc.Hint, err = encodePageValue(c.hint); err != nil {
		return nil, fmt.Errorf("paging hint: %w", err)
	}
	if c.collation != nil {
		if pc.Collation, err = encodePageDoc(c.collation); err != nil {
			return nil, fmt.Errorf("paging collation: %w", err)
		}
	}
	return pc, nil
}

// pageQuery holds a PageContext's values decoded back into ordered BSON.
type pageQuery struct {
	filter, projection, sort, hint any
	collation                      map[string]any
}

func decodePageContext(pc models.PageContext) (pageQuery, error) {
	var q pageQuery
	var err error
	if q.filter, err = decodePageDoc(pc.Filter); err != nil {
		return q, fmt.Errorf("paging filter: %w", err)
	}
	if q.projection, err = decodePageDoc(pc.Projection); err != nil {
		return q, fmt.Errorf("paging projection: %w", err)
	}
	if q.sort, err = decodePageDoc(pc.Sort); err != nil {
		return q, fmt.Errorf("paging sort: %w", err)
	}
	if q.hint, err = decodePageValue(pc.Hint); err != nil {
		return q, fmt.Errorf("paging hint: %w", err)
	}
	collation, err := decodePageDoc(pc.Collation)
	if err != nil {
		return q, fmt.Errorf("paging collation: %w", err)
	}
	if collation != nil {
		q.collation, _ = asMap(collation)
	}
	return q, nil
}

// effectivePaging computes the skip/limit to send to MongoDB for a given page.
// Returns empty=true when the page lies beyond pc.UserLimit (caller should
// short-circuit to an empty result without dispatching).
func effectivePaging(pc models.PageContext, page, pageSize int64) (skip, limit int64, empty bool) {
	skip = pc.UserSkip + page*pageSize
	if pc.UserLimit > 0 {
		remaining := pc.UserLimit - page*pageSize
		if remaining <= 0 {
			return 0, 0, true
		}
		if remaining < pageSize {
			return skip, remaining, false
		}
	}
	return skip, pageSize, false
}

// FetchPage runs a single-page find against MongoDB using a previously
// captured PageContext. Stateless: each call is a fresh dispatch.
func (e *GojaEngine) FetchPage(
	ctx context.Context,
	dbName string,
	pc models.PageContext,
	page, pageSize int64,
) (models.QueryResult, error) {
	skip, limit, empty := effectivePaging(pc, page, pageSize)
	if empty {
		return models.QueryResult{Documents: []any{}}, nil
	}
	q, err := decodePageContext(pc)
	if err != nil {
		return models.QueryResult{}, err
	}
	op := CapturedOp{
		Collection: pc.Collection,
		Method:     "find",
		Args:       []any{q.filter, q.projection},
		Limit:      limit,
		Skip:       skip,
		Sort:       q.sort,
		Hint:       q.hint,
		Collation:  q.collation,
		MaxTimeMS:  pc.MaxTimeMS,
		BatchSize:  pc.BatchSize,
		Comment:    pc.Comment,
	}
	return dispatch(ctx, e.client, dbName, op)
}

// CountForPage returns a total-row count for a PageContext. When the filter is
// empty/nil, returns estimatedDocumentCount with estimated=true. Otherwise
// returns countDocuments(filter), capped by pc.UserLimit if set.
func (e *GojaEngine) CountForPage(
	ctx context.Context,
	dbName string,
	pc models.PageContext,
) (count int64, estimated bool, err error) {
	filter, err := decodePageDoc(pc.Filter)
	if err != nil {
		return 0, false, fmt.Errorf("paging filter: %w", err)
	}
	method := "countDocuments"
	args := []any{filter}
	if isEmptyFilter(filter) {
		method = "estimatedDocumentCount"
		args = nil
		estimated = true
	}
	op := CapturedOp{
		Collection: pc.Collection,
		Method:     method,
		Args:       args,
	}
	res, err := dispatch(ctx, e.client, dbName, op)
	if err != nil {
		return 0, estimated, err
	}
	count = extractCount(res)
	if pc.UserLimit > 0 && count > pc.UserLimit {
		count = pc.UserLimit
	}
	return count, estimated, nil
}

func isEmptyFilter(f any) bool {
	if f == nil {
		return true
	}
	switch d := f.(type) {
	case bson.D:
		return len(d) == 0
	case map[string]any:
		return len(d) == 0
	}
	return false
}

func extractCount(r models.QueryResult) int64 {
	if len(r.Documents) == 0 {
		return 0
	}
	doc, ok := asMap(r.Documents[0])
	if !ok {
		return 0
	}
	return coerceInt64(doc["count"])
}

// coerceInt64 handles raw numeric values and ExtJSON-canonical wrappers
// such as {"$numberLong": "60"} or {"$numberInt": "60"} produced by
// dispatchCountDocuments / dispatchEstimatedDocumentCount round-tripping.
func coerceInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int32:
		return int64(x)
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		var n int64
		fmt.Sscanf(x, "%d", &n)
		return n
	case bson.D, map[string]any:
		m, _ := asMap(x)
		for _, key := range []string{"$numberLong", "$numberInt", "$numberDouble"} {
			if s, ok := m[key].(string); ok {
				var n int64
				fmt.Sscanf(s, "%d", &n)
				return n
			}
		}
	}
	return 0
}

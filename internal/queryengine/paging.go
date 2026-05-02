package queryengine

import (
	"context"

	"vervet/internal/models"
)

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
	op := CapturedOp{
		Collection: pc.Collection,
		Method:     "find",
		Args:       []any{pc.Filter, pc.Projection},
		Limit:      limit,
		Skip:       skip,
		Sort:       pc.Sort,
		Hint:       pc.Hint,
		Collation:  pc.Collation,
		MaxTimeMS:  pc.MaxTimeMS,
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
	method := "countDocuments"
	args := []any{pc.Filter}
	if isEmptyFilter(pc.Filter) {
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
	if m, ok := f.(map[string]any); ok && len(m) == 0 {
		return true
	}
	return false
}

func extractCount(r models.QueryResult) int64 {
	if len(r.Documents) == 0 {
		return 0
	}
	doc, ok := r.Documents[0].(map[string]any)
	if !ok {
		return 0
	}
	switch v := doc["count"].(type) {
	case int64:
		return v
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case int:
		return int64(v)
	}
	return 0
}

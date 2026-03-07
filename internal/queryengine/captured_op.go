package queryengine

// CapturedOp represents a MongoDB operation captured by the goja proxy layer.
// Instead of executing immediately, db.collection.method(...) calls are captured
// as CapturedOp values so they can be dispatched to the real MongoDB driver.
type CapturedOp struct {
	Collection string
	Method     string
	Args       []any
}

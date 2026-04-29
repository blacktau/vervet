package schema

type fieldNode struct {
	name     string
	count    int
	types    map[string]*typeStat
	children map[string]*fieldNode
}

type typeStat struct {
	name   string
	count  int
	minNum *float64
	maxNum *float64
	minStr *string
	maxStr *string
	minLen *int
	maxLen *int
}

type accumulator struct {
	root map[string]*fieldNode
}

func newAccumulator() *accumulator {
	return &accumulator{root: make(map[string]*fieldNode)}
}

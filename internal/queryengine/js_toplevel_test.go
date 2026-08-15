package queryengine

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteTopLevel_ConstAndLetBecomeVar(t *testing.T) {
	got := rewriteTopLevelDeclarations("const a = 1;\nlet b = 2;")
	assert.Equal(t, "var   a = 1;\nvar b = 2;", got)
}

// Line and column numbers in runtime errors have to keep pointing at the user's
// source, so the rewrite must not change the length of the script.
func TestRewriteTopLevel_PreservesLength(t *testing.T) {
	src := "const a = 1;\nlet b = 2;\nconst c = [1, 2];"
	assert.Len(t, rewriteTopLevelDeclarations(src), len(src))
}

func TestRewriteTopLevel_LeavesNestedDeclarationsAlone(t *testing.T) {
	cases := map[string]string{
		"in a function":  "function f() { const a = 1; }",
		"in a block":     "{ let a = 1; }",
		"in a loop head": "for (const x of xs) { f(x); }",
		"in a string":    "print('const a = 1');",
		"in a comment":   "// const a = 1\nf();",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, src, rewriteTopLevelDeclarations(src))
		})
	}
}

// The reason the rewrite exists: mongosh runs scripts at top level, where
// `const db = db.getSiblingDB(...)` reads the existing global on the right.
func TestRewriteTopLevel_AllowsSelfReferentialGlobalRebind(t *testing.T) {
	rt := goja.New()
	require.NoError(t, rt.Set("db", map[string]any{
		"getSiblingDB": func(name string) string { return "sibling:" + name },
	}))

	_, err := rt.RunString(`const db = db.getSiblingDB("other");`)
	require.Error(t, err, "plain ECMAScript should hit the temporal dead zone")

	rt2 := goja.New()
	require.NoError(t, rt2.Set("db", map[string]any{
		"getSiblingDB": func(name string) string { return "sibling:" + name },
	}))
	val, err := rt2.RunString(rewriteTopLevelDeclarations(`const db = db.getSiblingDB("other"); db`))
	require.NoError(t, err)
	assert.Equal(t, "sibling:other", val.Export())
}

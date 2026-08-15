package shell

import (
	"strings"
	"testing"
)

// A script must stay at top level. Wrapping it in a function changes the
// meaning of its declarations: `const db = db.getSiblingDB(name)` reads the
// existing global at top level, but throws inside a function body.
func TestWrapQuery_DoesNotWrapScriptInAFunction(t *testing.T) {
	got := wrapQuery(`const db = db.getSiblingDB("other");`)
	if strings.Contains(got, "(() => {") {
		t.Errorf("script must not be wrapped in an IIFE; got:\n%s", got)
	}
	if !strings.Contains(got, `const db = db.getSiblingDB("other");`) {
		t.Errorf("script body altered; got:\n%s", got)
	}
}

func TestWrapQuery_CapturesLastExpression(t *testing.T) {
	got := wrapQuery(`db.users.find({})`)
	if !strings.Contains(got, resultGlobal+" = db.users.find({})") {
		t.Errorf("expected the final expression to be captured; got:\n%s", got)
	}
}

// A script whose last statement is a declaration has no value to print, and
// printing "null" for it adds a bogus line to the script's own output.
func TestWrapQuery_CapturesNothingForDeclarations(t *testing.T) {
	got := wrapQuery("print('done');\nconst x = 1;")
	if strings.Contains(got, resultGlobal+" = const") {
		t.Errorf("declaration must not be captured; got:\n%s", got)
	}
	if !strings.Contains(got, "if (__result === undefined) { return; }") {
		t.Errorf("expected the undefined guard in the epilogue; got:\n%s", got)
	}
}

// Scripts commonly end in a comment block. The capture has to attach to the
// last real statement, not to the comment (which is a syntax error).
func TestWrapQuery_IgnoresTrailingComments(t *testing.T) {
	src := "db.users.find({})\n// closing note\n// more notes"
	got := wrapQuery(src)
	if !strings.Contains(got, resultGlobal+" = db.users.find({})") {
		t.Errorf("expected capture on the last real statement; got:\n%s", got)
	}
}

func TestLastStatementStart_SkipsTrailingComments(t *testing.T) {
	src := "a();\nb();\n// trailing\n/* block */"
	got := src[lastStatementStart(src):]
	if !strings.HasPrefix(got, "b();") {
		t.Errorf("expected last statement to be b(); got %q", got)
	}
}

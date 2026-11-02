package shell

import (
	"strings"
	"testing"
)

// A script must stay at top level. Wrapping it in a function changes the
// meaning of its declarations: `const db = db.getSiblingDB(name)` reads the
// existing global at top level, but throws inside a function body.
func TestWrapQuery_DoesNotWrapScriptInAFunction(t *testing.T) {
	got, _ := wrapQuery(`const db = db.getSiblingDB("other");`, "n")
	lines := strings.Split(got, "\n")
	if !strings.HasSuffix(lines[0], "})();") {
		t.Errorf("the preamble must close before the script; got:\n%s", lines[0])
	}
	if lines[1] != `const db = db.getSiblingDB("other");` {
		t.Errorf("the script must be on its own line, unaltered; got:\n%s", lines[1])
	}
}

func TestWrapQuery_CapturesLastExpression(t *testing.T) {
	got, _ := wrapQuery(`db.users.find({})`, "n")
	if !strings.Contains(got, resultGlobal+" = db.users.find({})") {
		t.Errorf("expected the final expression to be captured; got:\n%s", got)
	}
}

// A script whose last statement is a declaration has no value to print, and
// printing "null" for it adds a bogus line to the script's own output.
func TestWrapQuery_CapturesNothingForDeclarations(t *testing.T) {
	got, _ := wrapQuery("print('done');\nconst x = 1;", "n")
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
	got, _ := wrapQuery(src, "n")
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

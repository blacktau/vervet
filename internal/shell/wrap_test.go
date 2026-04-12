package shell

import (
	"strings"
	"testing"
)

func TestPrependReturn_SingleLineExpression(t *testing.T) {
	got := prependReturnToLastStatement(`db.users.find({})`)
	want := `return db.users.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_MultilineExpression(t *testing.T) {
	src := `db.createRole(
  {
    role: "x",
    privileges: []
  }
)`
	got := prependReturnToLastStatement(src)
	if !strings.HasPrefix(got, "return db.createRole(") {
		t.Errorf("expected return prepended to full createRole call, got: %q", got)
	}
	if !strings.Contains(got, `role: "x"`) {
		t.Errorf("body lost; got: %q", got)
	}
}

func TestPrependReturn_MongoshDirectiveFollowedByExpression(t *testing.T) {
	src := `use admin

db.createRole(
  { role: "x", privileges: [], roles: [] }
)`
	got := prependReturnToLastStatement(src)
	if !strings.Contains(got, "use admin") {
		t.Errorf("use directive stripped: %q", got)
	}
	if !strings.Contains(got, "return db.createRole(") {
		t.Errorf("return not prepended to last statement: %q", got)
	}
}

func TestPrependReturn_LeadingComments(t *testing.T) {
	src := `// header comment
// another
db.users.find({})`
	got := prependReturnToLastStatement(src)
	want := `// header comment
// another
return db.users.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_AlreadyHasReturn(t *testing.T) {
	src := `return db.x.find({})`
	got := prependReturnToLastStatement(src)
	if got != src {
		t.Errorf("should not double-prepend; got %q", got)
	}
}

func TestPrependReturn_ClosingBraceLeftAlone(t *testing.T) {
	src := `if (true) {
  db.x.find({})
}`
	got := prependReturnToLastStatement(src)
	if got != src {
		t.Errorf("should not prepend return to closing brace; got %q", got)
	}
}

func TestPrependReturn_ControlFlowKeywords(t *testing.T) {
	cases := []string{
		`const x = 1`,
		`let y = 2`,
		`var z = 3`,
		`if (x) { doThing() }`,
		`for (const i of xs) { f(i) }`,
		`function foo() { return 1 }`,
	}
	for _, src := range cases {
		got := prependReturnToLastStatement(src)
		if got != src {
			t.Errorf("should not prepend return before keyword; src=%q got=%q", src, got)
		}
	}
}

func TestPrependReturn_StringWithBrackets(t *testing.T) {
	src := `db.x.find({ name: "a)b(c" })`
	got := prependReturnToLastStatement(src)
	want := `return db.x.find({ name: "a)b(c" })`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_SemicolonSeparatedStatements(t *testing.T) {
	src := `var a = 1; db.x.find({})`
	got := prependReturnToLastStatement(src)
	want := `var a = 1; return db.x.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_Empty(t *testing.T) {
	got := prependReturnToLastStatement("")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestPrependReturn_CommentWithBrackets(t *testing.T) {
	src := `// example: db.x.find({ a: 1 })
db.y.find({})`
	got := prependReturnToLastStatement(src)
	want := `// example: db.x.find({ a: 1 })
return db.y.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_TrailingBlankLines(t *testing.T) {
	src := "db.x.find({})\n\n\n"
	got := prependReturnToLastStatement(strings.TrimSpace(src))
	want := `return db.x.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

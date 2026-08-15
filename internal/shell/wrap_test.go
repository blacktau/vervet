package shell

import (
	"strings"
	"testing"
)

func TestPrependReturn_SingleLineExpression(t *testing.T) {
	got := prependToLastStatement(`db.users.find({})`, "return ")
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
	got := prependToLastStatement(src, "return ")
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
	got := prependToLastStatement(src, "return ")
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
	got := prependToLastStatement(src, "return ")
	want := `// header comment
// another
return db.users.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_AlreadyHasReturn(t *testing.T) {
	src := `return db.x.find({})`
	got := prependToLastStatement(src, "return ")
	if got != src {
		t.Errorf("should not double-prepend; got %q", got)
	}
}

func TestPrependReturn_ClosingBraceLeftAlone(t *testing.T) {
	src := `if (true) {
  db.x.find({})
}`
	got := prependToLastStatement(src, "return ")
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
		got := prependToLastStatement(src, "return ")
		if got != src {
			t.Errorf("should not prepend return before keyword; src=%q got=%q", src, got)
		}
	}
}

func TestPrependReturn_StringWithBrackets(t *testing.T) {
	src := `db.x.find({ name: "a)b(c" })`
	got := prependToLastStatement(src, "return ")
	want := `return db.x.find({ name: "a)b(c" })`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_SemicolonSeparatedStatements(t *testing.T) {
	src := `var a = 1; db.x.find({})`
	got := prependToLastStatement(src, "return ")
	want := `var a = 1; return db.x.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_Empty(t *testing.T) {
	got := prependToLastStatement("", "return ")
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestPrependReturn_CommentWithBrackets(t *testing.T) {
	src := `// example: db.x.find({ a: 1 })
db.y.find({})`
	got := prependToLastStatement(src, "return ")
	want := `// example: db.x.find({ a: 1 })
return db.y.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrependReturn_TrailingBlankLines(t *testing.T) {
	src := "db.x.find({})\n\n\n"
	got := prependToLastStatement(strings.TrimSpace(src), "return ")
	want := `return db.x.find({})`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

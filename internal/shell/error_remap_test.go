package shell

import (
	"strings"
	"testing"
)

func TestRemapError_MultiLineCreateRole(t *testing.T) {
	userQuery := `use admin

db.createRole(
   {
     role: "myClusterwideAdmin",
     privileges: [],
     roles: [
       { role: "read", db: "admin" }
     ]
   }
)`
	// User query is 11 lines; wrapper adds leading empty line, so line 12
	// of the wrapped script is user line 11 (the closing paren).
	mongoshErr := `SyntaxError: Unexpected token, expected "," (12:0)

  10 |    }
  11 |    }
> 12 | return ) })();
     | ^
  13 | const __val = (typeof __result?.toArray === 'function') ? __result.toArray() : __result;`

	got := remap(mongoshErr, userQuery)

	if strings.Contains(got, "})();") {
		t.Errorf("wrapper trailer should be stripped; got:\n%s", got)
	}
	if strings.Contains(got, "__val") {
		t.Errorf("wrapper lines should be stripped; got:\n%s", got)
	}
	if !strings.Contains(got, "> 11 | )") {
		t.Errorf("expected remapped highlight on user line 11 (closing paren); got:\n%s", got)
	}
	if !strings.Contains(got, "10 |    }") {
		t.Errorf("expected context line 10 remapped to user source; got:\n%s", got)
	}
}

// caretColumn is how far the caret sits after the pipe in got's caret line.
func caretColumn(t *testing.T, got string) int {
	t.Helper()
	caretLine := findLineContaining(got, "^")
	if caretLine == "" {
		t.Fatalf("caret line missing; got:\n%s", got)
	}
	return strings.IndexByte(caretLine, '^') - strings.IndexByte(caretLine, '|')
}

// The line holding the capture prefix is shown as the user wrote it, so the
// caret and the (line:col) header move left by the prefix's length. These are
// mongosh's real reports for each query, wrapped by wrapQuery.
func TestRemapError_CaptureLineShiftsCaretAndHeader(t *testing.T) {
	userQuery := "foo(1,;"
	mongoshErr := `SyntaxError: Unexpected token (2:34)

> 2 | globalThis.__vervetResult = foo(1,;
    |                                   ^`

	got := remap(mongoshErr, userQuery)

	if !strings.Contains(got, "SyntaxError: Unexpected token (1:6)") {
		t.Errorf("expected header at user line 1, column 6; got:\n%s", got)
	}
	if !strings.Contains(got, "> 1 | foo(1,;") {
		t.Errorf("expected user line 1 with original source; got:\n%s", got)
	}
	if col := caretColumn(t, got); col != 8 {
		t.Errorf("caret should sit under the ';' (column 8 after the pipe), got %d:\n%s", col, got)
	}
}

func TestRemapError_CaptureOnLaterLine(t *testing.T) {
	userQuery := "print(1)\nfoo(1,;"
	mongoshErr := `SyntaxError: Unexpected token (3:34)

  2 | print(1)
> 3 | globalThis.__vervetResult = foo(1,;
    |                                   ^`

	got := remap(mongoshErr, userQuery)

	if !strings.Contains(got, "(2:6)") {
		t.Errorf("expected header at user line 2, column 6; got:\n%s", got)
	}
	if col := caretColumn(t, got); col != 8 {
		t.Errorf("caret should sit under the ';', got %d:\n%s", col, got)
	}
}

// A line without the capture prefix keeps its column.
func TestRemapError_NoCaptureKeepsColumn(t *testing.T) {
	userQuery := "let x = (;"
	mongoshErr := `SyntaxError: Unexpected token (2:9)

> 2 | let x = (;
    |          ^`

	got := remap(mongoshErr, userQuery)

	if !strings.Contains(got, "(1:9)") {
		t.Errorf("expected header at user line 1, column 9; got:\n%s", got)
	}
	if col := caretColumn(t, got); col != 11 {
		t.Errorf("caret should stay under the ';', got %d:\n%s", col, got)
	}
}

func TestRemapError_PlainStderrUnchanged(t *testing.T) {
	err := `MongoServerError: not authorized on admin to execute command`
	got := remap(err, `db.whatever()`)
	if got != err {
		t.Errorf("non-trace error should pass through; got %q", got)
	}
}

func TestRemapError_EmptyUserQuery(t *testing.T) {
	got := remap("anything", "")
	if got != "anything" {
		t.Errorf("empty user query should return input unchanged; got %q", got)
	}
}

func TestRemapError_ContextLinesBeyondUserSource(t *testing.T) {
	userQuery := `db.x.find({})`
	mongoshErr := `SyntaxError: boom (2:26)
> 2 | const __result = (() => { db.x.find({}) })();
  3 | const __val = ...
  4 | try {`
	got := remap(mongoshErr, userQuery)
	if strings.Contains(got, "__val") {
		t.Errorf("wrapper context lines should be dropped; got:\n%s", got)
	}
	if strings.Contains(got, "try {") {
		t.Errorf("wrapper context lines should be dropped; got:\n%s", got)
	}
	if !strings.Contains(got, "> 1 | db.x.find({})") {
		t.Errorf("expected user line 1; got:\n%s", got)
	}
}

// remap runs remapError with the capture position wrapQuery reports for query.
func remap(errMsg, query string) string {
	_, capture := wrapQuery(query, "n")
	return remapError(errMsg, query, capture)
}

func findLineContaining(s, substr string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

// A warning on stderr can come before the error, so the (line:col) is not
// always on the first line.
func TestRemapError_PositionOnALaterLine(t *testing.T) {
	userQuery := "let x = (;"
	mongoshErr := "Warning: something deprecated\nSyntaxError: Unexpected token (2:9)\n\n> 2 | let x = (;\n    |          ^"

	got := remap(mongoshErr, userQuery)

	if !strings.Contains(got, "SyntaxError: Unexpected token (1:9)") {
		t.Errorf("expected the position remapped on the second line; got:\n%s", got)
	}
}

// The caret on the capture line only moves when it points past the prefix.
// mongosh's real report for `x = {a:,}; 5`: the error is before the prefix.
func TestRemapError_CaretBeforeCapturePrefixStays(t *testing.T) {
	userQuery := "x = {a:,}; 5"
	mongoshErr := "SyntaxError: Unexpected token (2:7)\n\n" +
		"  1 | ;(function () {})();\n" +
		"> 2 | x = {a:,}; globalThis.__vervetResult = 5\n" +
		"    |        ^\n" +
		"  3 |"

	got := remap(mongoshErr, userQuery)

	if !strings.Contains(got, "SyntaxError: Unexpected token (1:7)") {
		t.Errorf("expected header at user line 1, column 7; got:\n%s", got)
	}
	if col := caretColumn(t, got); col != 9 {
		t.Errorf("caret should stay under the ',' (column 9 after the pipe), got %d:\n%s", col, got)
	}
}

// Only mongosh's code frame is remapped. Text the script wrote, or a server
// error, that ends in "(N:M)" is left as it is.
func TestRemapError_LeavesOtherPositionsAlone(t *testing.T) {
	errMsg := "ratio (3:4)\nMongoServerError: bad value at (2:5)"
	if got := remap(errMsg, "print('x')\ndb.c.find()"); got != errMsg {
		t.Errorf("text outside a code frame should pass through; got:\n%s", got)
	}
}

// Lines after the code frame are not part of it, even if they look numbered.
func TestRemapError_StopsAtTheEndOfTheFrame(t *testing.T) {
	userQuery := "let x = (;"
	mongoshErr := "SyntaxError: Unexpected token (2:9)\n\n> 2 | let x = (;\n    |          ^\n\nShell cwd was reset\n  7 | not a frame line"

	got := remap(mongoshErr, userQuery)

	if !strings.HasSuffix(got, "\n\nShell cwd was reset\n  7 | not a frame line") {
		t.Errorf("lines after the frame should be unchanged; got:\n%s", got)
	}
}

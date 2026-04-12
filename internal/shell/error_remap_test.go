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
	mongoshErr := `SyntaxError: Unexpected token, expected ","

  10 |    }
  11 |    }
> 12 | return ) })();
     | ^
  13 | const __val = (typeof __result?.toArray === 'function') ? __result.toArray() : __result;`

	got := remapError(mongoshErr, userQuery)

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

func TestRemapError_FirstLineErrorAdjustsColumn(t *testing.T) {
	userQuery := `db.x.find({ bad: })`
	mongoshErr := `SyntaxError: Unexpected token
> 2 | const __result = (() => { db.x.find({ bad: }) })();
    |                                             ^`

	got := remapError(mongoshErr, userQuery)

	if !strings.Contains(got, "> 1 | db.x.find({ bad: })") {
		t.Errorf("expected user line 1 with original source; got:\n%s", got)
	}
	caretLine := findLineContaining(got, "^")
	if caretLine == "" {
		t.Fatalf("caret line missing; got:\n%s", got)
	}
	pipeIdx := strings.IndexByte(caretLine, '|')
	caretIdx := strings.IndexByte(caretLine, '^')
	if caretIdx-pipeIdx >= 30 {
		t.Errorf("caret column not shifted left; got caret line %q", caretLine)
	}
}

func TestRemapError_PlainStderrUnchanged(t *testing.T) {
	err := `MongoServerError: not authorized on admin to execute command`
	got := remapError(err, `db.whatever()`)
	if got != err {
		t.Errorf("non-trace error should pass through; got %q", got)
	}
}

func TestRemapError_EmptyUserQuery(t *testing.T) {
	got := remapError("anything", "")
	if got != "anything" {
		t.Errorf("empty user query should return input unchanged; got %q", got)
	}
}

func TestRemapError_ContextLinesBeyondUserSource(t *testing.T) {
	userQuery := `db.x.find({})`
	mongoshErr := `SyntaxError: boom
> 2 | const __result = (() => { db.x.find({}) })();
  3 | const __val = ...
  4 | try {`
	got := remapError(mongoshErr, userQuery)
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

func findLineContaining(s, substr string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	return ""
}

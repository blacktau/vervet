package queryengine

// mongosh evaluates a script's top-level declarations as properties of the
// global object, so `const db = db.getSiblingDB("other")` reads the existing
// global `db` on the right-hand side and rebinds it. Plain ECMAScript puts that
// declaration in the temporal dead zone instead, and the same line throws
// "Cannot access 'db' before initialization".
//
// Rewriting top-level const/let to var restores the mongosh behaviour: `var`
// redeclaring an existing global property does not reset it, so the right-hand
// side still sees the original value. Declarations inside functions, blocks and
// loop headers are left alone, so block scoping is unaffected.
//
// The replacement is length-preserving ("const" -> "var  ", "let" -> "var")
// so error line and column numbers still point at the user's source.
func rewriteTopLevelDeclarations(src string) string {
	out := []byte(src)
	depth := 0
	atStatementStart := true
	i := 0
	n := len(out)

	for i < n {
		c := out[i]

		switch {
		case c == '/' && i+1 < n && out[i+1] == '/':
			for i < n && out[i] != '\n' {
				i++
			}
			continue
		case c == '/' && i+1 < n && out[i+1] == '*':
			i += 2
			for i+1 < n && !(out[i] == '*' && out[i+1] == '/') {
				i++
			}
			i += 2
			continue
		case c == '\'' || c == '"' || c == '`':
			quote := c
			i++
			for i < n && out[i] != quote {
				if out[i] == '\\' {
					i++
				}
				i++
			}
			i++
			atStatementStart = false
			continue
		}

		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}

		if depth == 0 && atStatementStart {
			if kw := declarationKeywordAt(out, i); kw != 0 {
				copy(out[i:], "var")
				for pad := 3; pad < kw; pad++ {
					out[i+pad] = ' '
				}
				i += kw
				atStatementStart = false
				continue
			}
		}

		switch c {
		case '(', '[', '{':
			depth++
			atStatementStart = true
		case ')', ']':
			if depth > 0 {
				depth--
			}
			atStatementStart = false
		case '}':
			if depth > 0 {
				depth--
			}
			atStatementStart = true
		case ';', '\n':
			atStatementStart = true
		default:
			atStatementStart = false
		}
		i++
	}

	return string(out)
}

// declarationKeywordAt returns the length of a const/let keyword starting at i
// when it is followed by whitespace, or 0 when there is no declaration there.
func declarationKeywordAt(src []byte, i int) int {
	for _, kw := range []string{"const", "let"} {
		end := i + len(kw)
		if end < len(src) && string(src[i:end]) == kw && isJSSpace(src[end]) {
			return len(kw)
		}
	}
	return 0
}

func isJSSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

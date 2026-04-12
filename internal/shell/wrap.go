package shell

import "strings"

// skipKeywords are statement-starting tokens where prepending "return " would
// be a syntax error or change meaning. When the last top-level statement
// begins with one of these, we leave it alone.
var skipKeywords = []string{
	"return", "if", "else", "for", "while", "do", "switch", "case",
	"break", "continue", "throw", "try", "catch", "finally",
	"function", "class", "const", "let", "var", "import", "export",
	"use ",
}

// prependReturnToLastStatement inserts "return " at the start of the final
// top-level statement in src. Bracket depth, strings, and comments are
// tracked so multi-line expressions and strings containing semicolons don't
// confuse the scan. If the final statement begins with a skip keyword or a
// closing brace, the source is returned unchanged.
func prependReturnToLastStatement(src string) string {
	if src == "" {
		return src
	}

	start := lastStatementStart(src)
	stmt := src[start:]
	if stmt == "" {
		return src
	}
	if stmt[0] == '}' || stmt[0] == '{' {
		return src
	}
	for _, kw := range skipKeywords {
		if hasKeywordPrefix(stmt, kw) {
			return src
		}
	}

	return src[:start] + "return " + stmt
}

// hasKeywordPrefix reports whether s begins with kw followed by a
// non-identifier character (or end of string). Ensures we don't match
// "returnSomething" when looking for "return".
func hasKeywordPrefix(s, kw string) bool {
	if !strings.HasPrefix(s, kw) {
		return false
	}
	if len(s) == len(kw) {
		return true
	}
	c := s[len(kw)]
	if kw[len(kw)-1] == ' ' {
		return true
	}
	return !isIdentChar(c)
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

// lastStatementStart returns the byte offset of the last top-level statement
// in src. A statement boundary is a semicolon or newline at bracket depth 0,
// outside any string or comment.
func lastStatementStart(src string) int {
	lastBoundary := 0
	depth := 0
	i := 0
	n := len(src)

	for i < n {
		c := src[i]

		// Line comment
		if c == '/' && i+1 < n && src[i+1] == '/' {
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment
		if c == '/' && i+1 < n && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
			continue
		}
		// Strings (single, double, backtick)
		if c == '\'' || c == '"' || c == '`' {
			quote := c
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
			if i < n {
				i++
			}
			continue
		}

		switch c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case ';', '\n':
			if depth == 0 {
				lastBoundary = i + 1
			}
		}
		i++
	}

	// Advance past any additional whitespace-only / empty lines after the
	// boundary so we don't end up pointing at blank lines.
	for lastBoundary < n {
		c := src[lastBoundary]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		lastBoundary++
	}
	if lastBoundary > n {
		lastBoundary = n
	}
	return lastBoundary
}

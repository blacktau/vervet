package shell

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// wrapperPrefixOnLine1 is the text the wrapper injects before the first
// line of the user's body. Error columns on wrapped line 2 (user line 1)
// need to subtract this length.
const wrapperPrefixOnLine1 = "const __result = (() => { "

// lineNumberRE matches a source-snippet line in a mongosh error trace, e.g.
// `  15 |      ]` or `> 17 | return ) })();`. Captures the leading
// whitespace, optional `>` marker, line number, separator, and source text.
var lineNumberRE = regexp.MustCompile(`^(\s*)(>?\s*)(\d+)(\s*\|\s*)(.*)$`)

// caretLineRE matches the caret indicator line, e.g. `     | ^`. Captures
// the leading column (spaces) and the caret itself so we can adjust it
// when the preceding error line was on wrapped line 2.
var caretLineRE = regexp.MustCompile(`^(\s*\|\s*)(\^+)(.*)$`)

// remapError rewrites mongosh error output so line numbers and source
// snippets refer to the user's original query rather than the wrapped
// script. Lines that point into wrapper code (before the body or after
// the trailer) are dropped.
func remapError(errMsg, userQuery string) string {
	trimmed := strings.TrimSpace(userQuery)
	if trimmed == "" {
		return errMsg
	}
	userLines := strings.Split(trimmed, "\n")
	inLines := strings.Split(errMsg, "\n")

	out := make([]string, 0, len(inLines))
	lastWasHighlight := false
	lastWasOnFirstUserLine := false

	for _, line := range inLines {
		if m := lineNumberRE.FindStringSubmatch(line); m != nil {
			lineNum, err := strconv.Atoi(m[3])
			if err != nil {
				out = append(out, line)
				lastWasHighlight = false
				continue
			}
			userLine := lineNum - 1
			if userLine < 1 || userLine > len(userLines) {
				lastWasHighlight = false
				continue
			}
			isHighlight := strings.Contains(m[2], ">")
			out = append(out, fmt.Sprintf("%s%s%d | %s", m[1], m[2], userLine, userLines[userLine-1]))
			lastWasHighlight = isHighlight
			lastWasOnFirstUserLine = isHighlight && userLine == 1
			continue
		}

		if lastWasHighlight {
			if m := caretLineRE.FindStringSubmatch(line); m != nil {
				if lastWasOnFirstUserLine {
					out = append(out, adjustCaretColumn(line, m, len(wrapperPrefixOnLine1)))
				} else {
					out = append(out, line)
				}
				lastWasHighlight = false
				lastWasOnFirstUserLine = false
				continue
			}
		}

		out = append(out, line)
		lastWasHighlight = false
		lastWasOnFirstUserLine = false
	}

	return strings.Join(out, "\n")
}

// adjustCaretColumn shifts the caret left by offset columns. If the caret
// would end up at or before the pipe, it is clamped to the first column
// after the pipe.
func adjustCaretColumn(line string, m []string, offset int) string {
	prefix := m[1]
	caret := m[2]
	rest := m[3]

	// prefix is like "     | ". Find the pipe position, then count spaces
	// between the pipe and caret.
	pipeIdx := strings.IndexByte(prefix, '|')
	if pipeIdx < 0 {
		return line
	}
	spacesAfterPipe := len(prefix) - pipeIdx - 1
	newSpaces := spacesAfterPipe - offset
	if newSpaces < 1 {
		newSpaces = 1
	}
	return prefix[:pipeIdx+1] + strings.Repeat(" ", newSpaces) + caret + rest
}

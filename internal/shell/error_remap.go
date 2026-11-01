package shell

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// capturePrefix is what wrapQuery inserts before the script's last statement.
// Positions on that line, past its start, are shifted right by its length.
const capturePrefix = resultGlobal + " = "

// positionRE matches the (line:col) position mongosh puts at the end of a
// syntax error's first line, e.g. `SyntaxError: Unexpected token (2:9)`.
var positionRE = regexp.MustCompile(`\((\d+):(\d+)\)$`)

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
// the trailer) are dropped. capture is where wrapQuery put its capture
// prefix, which shifts positions on that line.
func remapError(errMsg, userQuery string, capture capturePos) string {
	trimmed := strings.TrimSpace(userQuery)
	if trimmed == "" {
		return errMsg
	}
	userLines := strings.Split(trimmed, "\n")
	inLines := strings.Split(errMsg, "\n")
	capLine, capCol := capture.line, capture.col

	out := make([]string, 0, len(inLines))
	lastWasHighlight := false
	lastWasOnCaptureLine := false

	for _, line := range inLines {
		// The (line:col) is usually on the first line, but a stderr warning or
		// stray stdout can come before it.
		line = remapPosition(line, len(userLines), capLine, capCol)
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
			lastWasOnCaptureLine = isHighlight && userLine == capLine
			continue
		}

		if lastWasHighlight {
			if m := caretLineRE.FindStringSubmatch(line); m != nil {
				if lastWasOnCaptureLine {
					out = append(out, adjustCaretColumn(line, m, len(capturePrefix)))
				} else {
					out = append(out, line)
				}
				lastWasHighlight = false
				lastWasOnCaptureLine = false
				continue
			}
		}

		out = append(out, line)
		lastWasHighlight = false
		lastWasOnCaptureLine = false
	}

	return strings.Join(out, "\n")
}

// remapPosition rewrites a trailing (line:col) from wrapped-script terms to
// the user's: the preamble line is taken off, and so is the capture prefix
// when the position lies past it.
func remapPosition(line string, userLineCount, capLine, capCol int) string {
	m := positionRE.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}
	wrappedLine, _ := strconv.Atoi(line[m[2]:m[3]])
	col, _ := strconv.Atoi(line[m[4]:m[5]])
	userLine := wrappedLine - 1
	if userLine < 1 || userLine > userLineCount {
		return line
	}
	if userLine == capLine && col >= capCol+len(capturePrefix) {
		col -= len(capturePrefix)
	}
	return line[:m[0]] + fmt.Sprintf("(%d:%d)", userLine, col)
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

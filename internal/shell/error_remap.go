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

// remapError rewrites mongosh's syntax error report so its position and code
// frame refer to the user's query rather than the wrapped script. Only the
// report itself is touched: the "(line:col)" header and the frame of numbered
// source lines and caret right below it. Anything else in errMsg (stderr
// notices, text the script wrote) passes through unchanged, even when it
// happens to end in "(N:M)". Frame lines that point into wrapper code are
// dropped. capture is where wrapQuery put its capture prefix, which shifts
// columns on that line.
//
// mongosh reports runtime errors without a position or stack (verified on
// mongosh 2.7.0), so a syntax error is the only report there is to remap.
func remapError(errMsg, userQuery string, capture capturePos) string {
	trimmed := strings.TrimSpace(userQuery)
	if trimmed == "" {
		return errMsg
	}
	inLines := strings.Split(errMsg, "\n")
	header := codeFrameHeader(inLines)
	if header < 0 {
		return errMsg
	}
	userLines := strings.Split(trimmed, "\n")

	out := make([]string, 0, len(inLines))
	out = append(out, inLines[:header]...)
	out = append(out, remapPosition(inLines[header], len(userLines), capture))

	i := header + 1
	if i < len(inLines) && strings.TrimSpace(inLines[i]) == "" {
		out = append(out, inLines[i])
		i++
	}

	highlightLine := 0 // the user line the last ">" frame line showed, 0 if none
	for ; i < len(inLines); i++ {
		line := inLines[i]
		if m := lineNumberRE.FindStringSubmatch(line); m != nil {
			lineNum, _ := strconv.Atoi(m[3])
			userLine := lineNum - 1
			highlightLine = 0
			if userLine < 1 || userLine > len(userLines) {
				continue
			}
			if strings.Contains(m[2], ">") {
				highlightLine = userLine
			}
			out = append(out, fmt.Sprintf("%s%s%d | %s", m[1], m[2], userLine, userLines[userLine-1]))
			continue
		}
		if m := caretLineRE.FindStringSubmatch(line); m != nil {
			if highlightLine > 0 && highlightLine == capture.line {
				line = adjustCaretColumn(line, m, capture.col)
			}
			highlightLine = 0
			out = append(out, line)
			continue
		}
		// The frame has ended; the rest is not part of the report.
		break
	}
	out = append(out, inLines[i:]...)

	return strings.Join(out, "\n")
}

// codeFrameHeader returns the index of the line that heads mongosh's code
// frame: a line ending in "(line:col)" whose next non-blank line is a numbered
// source line. It returns -1 when errMsg has no code frame.
func codeFrameHeader(lines []string) int {
	for i, line := range lines {
		if !positionRE.MatchString(line) {
			continue
		}
		next := i + 1
		if next < len(lines) && strings.TrimSpace(lines[next]) == "" {
			next++
		}
		if next < len(lines) && lineNumberRE.MatchString(lines[next]) {
			return i
		}
	}
	return -1
}

// shiftsColumn reports whether a 0-based column on the capture line lies past
// the inserted prefix, and so must move left by its length.
func shiftsColumn(col int, capture capturePos) bool {
	return col >= capture.col+len(capturePrefix)
}

// remapPosition rewrites a trailing (line:col) from wrapped-script terms to
// the user's: the preamble line is taken off, and so is the capture prefix
// when the position lies past it.
func remapPosition(line string, userLineCount int, capture capturePos) string {
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
	if userLine == capture.line && shiftsColumn(col, capture) {
		col -= len(capturePrefix)
	}
	return line[:m[0]] + fmt.Sprintf("(%d:%d)", userLine, col)
}

// adjustCaretColumn moves a caret on the capture line left by the prefix's
// length when it points past the prefix (capCol is where the prefix went). A
// caret before the prefix already points at the user's own column.
func adjustCaretColumn(line string, m []string, capCol int) string {
	prefix := m[1]
	caret := m[2]
	rest := m[3]

	// prefix is like "     | ". Find the pipe position, then count spaces
	// between the pipe and caret.
	pipeIdx := strings.IndexByte(prefix, '|')
	if pipeIdx < 0 {
		return line
	}
	// One space separates the pipe from the source, so the caret's column
	// in the source is one less than the spaces after the pipe.
	spacesAfterPipe := len(prefix) - pipeIdx - 1
	if !shiftsColumn(spacesAfterPipe-1, capturePos{col: capCol}) {
		return line
	}
	newSpaces := spacesAfterPipe - len(capturePrefix)
	return prefix[:pipeIdx+1] + strings.Repeat(" ", newSpaces) + caret + rest
}

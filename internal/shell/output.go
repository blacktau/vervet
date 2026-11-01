package shell

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"vervet/internal/models"
)

// A script's printed output and its result share mongosh's stdout. To tell
// them apart, the wrapped script frames every print/printjson/console call
// and the result with marker lines that carry a nonce made fresh for each
// run, so a script cannot write a marker by accident:
//
//	\x1eVV:<nonce>:B:<level>   a call starts (info, warning or error)
//	\x1eVV:<nonce>:E           the call ends
//	\x1eVV:<nonce>:R           the rest of stdout is the result
//
// Every marker ends its line, as print adds a newline, but it need not start
// one: process.stdout.write leaves the line open, so a marker can follow other
// text. Nested frames (a wrapped function calling another) are counted so only
// the outermost call becomes an entry.

// frameMarker is the prefix of every marker line for nonce.
func frameMarker(nonce string) string {
	return "\x1eVV:" + nonce + ":"
}

// newNonce returns a random token for one run's markers.
func newNonce() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate output marker: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// outputPreamble returns the single line that goes before the script. It
// replaces print, printjson and console.* with versions that frame each call.
// console.warn and console.error write through print, so their output is sure
// to be on stdout, in its place among the other lines. console.trace keeps
// mongosh's own, which adds the "Trace:" line and stack. The original print
// is kept as __vervetPrint for the epilogue, which must write the result
// unframed.
func outputPreamble(nonce string) string {
	return `;(function () { const M = "\x1eVV:` + nonce + `:", P = globalThis.print, J = globalThis.printjson, T = console.trace; ` +
		`const w = (l, f) => function (...a) { P(M + "B:" + l); try { return f.apply(this, a); } finally { P(M + "E"); } }; ` +
		`globalThis.print = w("info", P); globalThis.printjson = w("info", J); ` +
		`for (const [k, l] of [["log", "info"], ["info", "info"], ["debug", "info"], ["warn", "warning"], ["error", "error"]]) { console[k] = w(l, P); } ` +
		`console.trace = w("info", T); ` +
		`globalThis.__vervetPrint = P; })();`
}

// scannedLine is one entry of a run's stdout. framed is false for text written
// outside any print/console call.
type scannedLine struct {
	models.OutputLine
	framed bool
}

// splitOutput separates the script's framed calls from its result. Text
// outside any frame (process.stdout.write, mongosh notices) is kept as info,
// one entry per run of lines.
func splitOutput(stdout, nonce string) (output []models.OutputLine, result string, hasResult bool) {
	scanned, result, hasResult := scanOutput(stdout, nonce)
	for _, l := range scanned {
		output = append(output, l.OutputLine)
	}
	return output, result, hasResult
}

func scanOutput(stdout, nonce string) (out []scannedLine, result string, hasResult bool) {
	if stdout == "" {
		return nil, "", false
	}
	marker := frameMarker(nonce)
	lines := splitAtMarkers(strings.Split(strings.TrimSuffix(stdout, "\n"), "\n"), marker)

	var (
		depth int
		level string
		body  []string
		stray []string
	)
	flushStray := func() {
		if text := strings.TrimSpace(strings.Join(stray, "\n")); text != "" {
			out = append(out, scannedLine{OutputLine: models.OutputLine{Level: models.OutputInfo, Text: text}})
		}
		stray = nil
	}
	closeCall := func() {
		out = append(out, scannedLine{OutputLine: models.OutputLine{Level: level, Text: strings.Join(body, "\n")}, framed: true})
	}

	for i, line := range lines {
		tag, isMarker := strings.CutPrefix(line, marker)
		switch {
		case isMarker && tag == "R" && depth == 0:
			flushStray()
			return out, strings.TrimSpace(strings.Join(lines[i+1:], "\n")), true
		case isMarker && strings.HasPrefix(tag, "B:"):
			if depth == 0 {
				flushStray()
				level = outputLevel(strings.TrimPrefix(tag, "B:"))
				body = nil
			}
			depth++
		case isMarker && tag == "E" && depth > 0:
			depth--
			if depth == 0 {
				closeCall()
			}
		case depth > 0:
			body = append(body, line)
		default:
			stray = append(stray, line)
		}
	}

	// A script that fails during a call never closes its frame.
	if depth > 0 {
		closeCall()
	}
	flushStray()
	return out, "", false
}

// splitAtMarkers puts a marker that follows other text on the same line (after
// process.stdout.write) on a line of its own, so the scan only sees markers at
// the start of a line.
func splitAtMarkers(lines []string, marker string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if i := strings.Index(line, marker); i > 0 {
			out = append(out, line[:i], line[i:])
			continue
		}
		out = append(out, line)
	}
	return out
}

// outputLevel accepts only the levels the preamble writes.
func outputLevel(l string) string {
	switch l {
	case models.OutputWarning, models.OutputError:
		return l
	default:
		return models.OutputInfo
	}
}

// buildResult turns a successful run's stdout and stderr into a QueryResult:
// the framed calls become Output, and the text after the result marker is
// parsed as the result. mongosh's own stderr notices follow as a warning;
// the script's console.error output is already in stdout.
func buildResult(stdout, stderr, nonce string) models.QueryResult {
	output, resultText, hasResult := splitOutput(stdout, nonce)

	var result models.QueryResult
	if hasResult {
		result = parseResult(resultText)
		result.HasValue = true
	}
	if text := strings.TrimSpace(stderr); text != "" {
		output = append(output, models.OutputLine{Level: models.OutputWarning, Text: text})
	}
	result.Output = output
	return result
}

// failedRun builds the result and error text for a run mongosh exited from
// with an error. What the script printed before it failed is returned in
// Output; the error text is stderr plus any stdout written outside a frame,
// which is where mongosh reports the error.
func failedRun(stdout, stderr, nonce string) (models.QueryResult, string) {
	scanned, _, _ := scanOutput(stdout, nonce)

	var printed []models.OutputLine
	parts := []string{}
	if text := strings.TrimSpace(stderr); text != "" {
		parts = append(parts, text)
	}
	for _, l := range scanned {
		if l.framed {
			printed = append(printed, l.OutputLine)
			continue
		}
		parts = append(parts, l.Text)
	}
	return models.QueryResult{Output: printed}, strings.Join(parts, "\n")
}

package easyreader

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
)

// reflowFix returns a SuggestedFix rewrapping block text to s.MaxLineWidth, or nil if unchanged or not pure `//`.
// Boundary lines (directives, list items, code fences, markers) are preserved verbatim.
func reflowFix(s Settings, block commentBlock) *analysis.SuggestedFix {
	if len(block.lines) == 0 || !block.isSlashBlock() {
		return nil
	}
	lines := reflow(block, s)
	if lines == nil {
		return nil
	}
	var b strings.Builder
	for i, t := range lines {
		if i > 0 {
			b.WriteByte('\n')
			b.WriteString(block.lines[0].indent)
		}
		b.WriteString("//")
		b.WriteString(t)
	}
	return &analysis.SuggestedFix{
		Message: "reflow comment text",
		TextEdits: []analysis.TextEdit{{
			Pos: block.lines[0].pos, End: block.lines[len(block.lines)-1].end,
			NewText: []byte(b.String()),
		}},
	}
}

// reflow returns the block's comment texts rewrapped to s.MaxLineWidth,
// or nil when the output is identical to the input.
func reflow(block commentBlock, s Settings) []string {
	// TODO: make clearer.
	var out []string
	var para []string
	flush := func() {
		if len(para) == 0 {
			return
		}
		out = append(out, wrap(strings.Join(para, " "), s.MaxLineWidth)...)
		para = nil
	}
	for _, line := range block.lines {
		t := strings.TrimRight(line.text, " \t")
		if isBoundary(t, block.isDoc(), s) {
			flush()
			out = append(out, line.text)
			continue
		}
		para = append(para, strings.TrimSpace(line.text))
	}
	flush()

	if len(out) != len(block.lines) {
		return out
	}
	for i := range out {
		if out[i] != block.lines[i].text {
			return out
		}
	}
	return nil
}

// wrap greedily packs words into lines fitting width.
// Each emitted line is the text after `//`, including its leading space.
func wrap(text string, width int) []string {
	limit := max(width-1, 1)
	var lines []string
	cur := ""
	for w := range strings.FieldsSeq(text) {
		if cur == "" {
			cur = w
			continue
		}
		if utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) > limit {
			lines = append(lines, " "+cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	if cur != "" {
		lines = append(lines, " "+cur)
	}
	return lines
}

// isBoundary reports whether a comment line must not merge with its neighbors during reflow.
// Boundaries: blank lines (doc comments only), code fences, directives, list items, separators, `{`/`}` endings, exemption markers.
func isBoundary(t string, isDoc bool, s Settings) bool {
	if t == "" {
		return isDoc
	}
	lead := strings.TrimLeft(t, " \t")
	return strings.HasPrefix(t, "```") ||
		directiveRe.MatchString(lead) ||
		listItemRe.MatchString(lead) ||
		separatorRe.MatchString(t) ||
		strings.HasSuffix(t, "{") ||
		strings.HasSuffix(t, "}") ||
		lineHasMarker(t, s)
}

// lineHasMarker reports whether a comment line carries a block-length exemption marker.
func lineHasMarker(t string, s Settings) bool {
	lt := strings.ToLower(t)
	for _, m := range s.ExemptionMarkers {
		if strings.Contains(lt, strings.ToLower(m)) {
			return true
		}
	}
	return false
}

// checkEmptyComment reports a blank `//` line between non-empty comment lines.
// Doc comments are exempt: blank `//` is a legitimate paragraph break.
func checkEmptyComment(pass *analysis.Pass, block commentBlock, i int, fix *analysis.SuggestedFix) {
	if block.isDoc() || !block.isEmptyMidLine(i) {
		return
	}
	d := analysis.Diagnostic{
		Pos: block.lines[i].pos, End: block.lines[i].end,
		Message: "empty comment line inside comment block",
	}
	if fix != nil {
		d.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	pass.Report(d)
}

// isEmptyMidLine reports whether block.lines[i] is a blank line with non-empty lines on both sides.
func (b commentBlock) isEmptyMidLine(i int) bool {
	if strings.TrimSpace(b.lines[i].text) != "" {
		return false
	}
	before, after := false, false
	for j := range i {
		if strings.TrimSpace(b.lines[j].text) != "" {
			before = true
			break
		}
	}
	for j := i + 1; j < len(b.lines); j++ {
		if strings.TrimSpace(b.lines[j].text) != "" {
			after = true
			break
		}
	}
	return before && after
}

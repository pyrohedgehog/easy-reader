package easyreader

import (
	"fmt"
	"go/token"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/tools/go/analysis"
)

var (
	// Exemption predicates shared by the punctuation and reflow rules.
	listItemRe   = regexp.MustCompile(`^([-*+][ \t]|[0-9]+[.)][ \t])`)
	urlTailRe    = regexp.MustCompile(`https?://[^ \t]+$`)
	separatorRe  = regexp.MustCompile(`^[-*=~_. \t]+$`)
	structuralRe = regexp.MustCompile(`[{},);]$`)
	directiveRe  = regexp.MustCompile(`^(go:|\+build|line\b|export\b|nolint:|lint:)`)
)

// checkLineWidth reports comment text wider than limit characters.
// Directives are exempt; fix, when non-nil, reflows the containing block.
func checkLineWidth(pass *analysis.Pass, limit int, l commentLine, fix *analysis.SuggestedFix) {
	t := strings.TrimRight(l.text, " \t\r")
	if t == "" || isDirective(t) {
		return
	}
	// TODO: this should ideally ignore the indentation depth prior to the comments start.
	n := utf8.RuneCountInString(t)
	if n <= limit {
		return
	}
	d := analysis.Diagnostic{
		Pos: l.pos, End: l.end,
		Message: fmt.Sprintf("comment exceeds %d width (%d chars)", limit, n),
	}
	if fix != nil && !l.trailing {
		d.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	pass.Report(d)
}

// checkPunctuation reports standalone comment lines without terminal punctuation.
// The fix appends the first EndPunctuation character, so fixed lines satisfy the setting.
func checkPunctuation(pass *analysis.Pass, s Settings, l commentLine) {
	t := strings.TrimRight(l.text, " \t\r")
	if t == "" || shouldSkipPunct(t) {
		return
	}
	last, _ := utf8.DecodeLastRuneInString(t)
	if strings.ContainsRune(s.EndPunctuation, last) {
		return
	}
	d := analysis.Diagnostic{
		Pos: l.pos, End: l.end,
		Message: fmt.Sprintf("comment does not end with %s", s.EndPunctuation),
	}
	if l.fixPos != token.NoPos {
		if first, size := utf8.DecodeRuneInString(s.EndPunctuation); size > 0 {
			d.SuggestedFixes = []analysis.SuggestedFix{{
				Message: fmt.Sprintf("append %q", string(first)),
				TextEdits: []analysis.TextEdit{{
					Pos: l.fixPos, End: l.fixPos, NewText: []byte(string(first)),
				}},
			}}
		}
	}
	pass.Report(d)
}

// checkBlockLength reports comment blocks over limit lines; a negative limit disables it.
// fix, when non-nil, reflows the block (which may drop it under the limit).
func checkBlockLength(pass *analysis.Pass, limit int, block commentBlock, exempt bool, fix *analysis.SuggestedFix) {
	if exempt || limit < 0 || len(block.lines) <= limit {
		return
	}
	d := analysis.Diagnostic{
		Pos: block.lines[0].pos, End: block.lines[len(block.lines)-1].end,
		Message: fmt.Sprintf("comment block is %d lines (max %d); add exemption marker if justified",
			len(block.lines), limit),
	}
	if fix != nil {
		d.SuggestedFixes = []analysis.SuggestedFix{*fix}
	}
	pass.Report(d)
}

// strangeChars are banned Unicode sequences: invisible or ambiguous in fixed-width fonts.
var strangeChars = []struct {
	seq  string
	name string
}{
	{"\u2013", "en dash"}, {"\u2014", "em dash"}, {"\u2012", "figure dash"},
	{"\u2015", "horizontal bar"}, {"\u2010", "hyphen"}, {"\u2011", "non-breaking hyphen"},
	{"\u2018", "left single quote"}, {"\u2019", "right single quote"},
	{"\u201c", "left double quote"}, {"\u201d", "right double quote"},
	{"\u00a0", "non-breaking space"}, {"\u2009", "thin space"}, {"\u200a", "hair space"},
	{"\u200b", "zero-width space"}, {"\u202f", "narrow no-break space"}, {"\u1680", "ogham space"},
	{"\u2026", "ellipsis"}, {"\u00ad", "soft hyphen"}, {"\ufeff", "BOM"},
	{"\u2028", "line separator"}, {"\u2029", "paragraph separator"},
}

// checkStrangeChars reports banned characters, once per kind per line.
func checkStrangeChars(pass *analysis.Pass, l commentLine) {
	for _, sc := range strangeChars {
		if strings.Contains(l.text, sc.seq) {
			pass.Reportf(l.pos, "comment contains %s", sc.name)
		}
	}
}

// shouldSkipPunct reports whether a standalone comment line is exempt from the punctuation rule:
// code fences, list items, URLs, separator lines, directives, and structural endings.
func shouldSkipPunct(t string) bool {
	lead := strings.TrimLeft(t, " \t")
	return strings.HasPrefix(t, "```") ||
		listItemRe.MatchString(lead) ||
		urlTailRe.MatchString(t) ||
		separatorRe.MatchString(t) ||
		structuralRe.MatchString(t) ||
		directiveRe.MatchString(lead)
}

// isDirective reports whether a comment line is a toolchain directive that must stay on one line.
func isDirective(t string) bool {
	return directiveRe.MatchString(strings.TrimLeft(t, " \t"))
}

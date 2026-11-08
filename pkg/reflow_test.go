package easyreader

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/tools/go/analysis"
)

func cl(text string) commentLine {
	return commentLine{text: text, slash: true, indent: "\t"}
}

func cb(doc docKind, lines ...commentLine) commentBlock {
	return commentBlock{lines: lines, meta: commentMeta{doc: doc}}
}

func TestReflowMergesShortContinuation(t *testing.T) {
	s := DefaultSettings()
	r := []commentLine{
		cl(" NewAnalyzer returns an analyzer configured by s (defaults applied to"),
		cl(" unset fields). Module plugins call this from BuildAnalyzers."),
	}
	got := reflow(cb(docFunc, r...), s)
	assert.Equal(t, []string{
		" NewAnalyzer returns an analyzer configured by s (defaults applied to unset fields). Module plugins call this from BuildAnalyzers.",
	}, got)
}

func TestReflowWrapsToWidth(t *testing.T) {
	s := DefaultSettings()
	s.MaxLineWidth = 20
	r := []commentLine{cl(" one two three four five six seven")}
	got := reflow(cb(docNone, r...), s)
	assert.Equal(t, []string{" one two three four", " five six seven"}, got)
}

func TestReflowPreservesBoundaries(t *testing.T) {
	s := DefaultSettings()
	r := []commentLine{
		cl(" items:"),
		cl(" - first"),
		cl(" - second"),
	}
	assert.Nil(t, reflow(cb(docNone, r...), s), "want nil (unchanged)")
}

func TestReflowCollapsesBlankLine(t *testing.T) {
	s := DefaultSettings()
	r := []commentLine{cl(" wall of text,"), cl(""), cl(" continued here.")}
	got := reflow(cb(docNone, r...), s)
	assert.Equal(t, []string{" wall of text, continued here."}, got)
}

func TestReflowPreservesBlankLineInDoc(t *testing.T) {
	s := DefaultSettings()
	r := []commentLine{cl(" first paragraph."), cl(""), cl(" second paragraph.")}
	assert.Nil(t, reflow(cb(docFunc, r...), s), "blank line preserved")
}

func TestReflowPreservesBoundaryWhitespace(t *testing.T) {
	s := DefaultSettings()
	r := []commentLine{
		cl(" - first  "),
		cl(" - second"),
	}
	assert.Nil(t, reflow(cb(docNone, r...), s), "boundary trailing whitespace preserved verbatim")
}

func TestReflowFixText(t *testing.T) {
	s := DefaultSettings()
	r := []commentLine{
		{pos: token.Pos(10), end: token.Pos(40), text: " short", slash: true, indent: "\t"},
		{pos: token.Pos(45), end: token.Pos(60), text: " tail.", slash: true, indent: "\t"},
	}
	fix := reflowFix(s, cb(docNone, r...))
	require.NotNil(t, fix)
	require.Len(t, fix.TextEdits, 1)
	e := fix.TextEdits[0]
	assert.Equal(t, token.Pos(10), e.Pos)
	assert.Equal(t, token.Pos(60), e.End)
	assert.Equal(t, "// short tail.", string(e.NewText))
}

func TestIsEmptyMidLine(t *testing.T) {
	b := cb(docNone, cl(" text,"), cl(""), cl(" more."))
	assert.True(t, b.isEmptyMidLine(1))
	assert.False(t, b.isEmptyMidLine(0))
	b2 := cb(docNone, cl(" text,"), cl(""))
	assert.False(t, b2.isEmptyMidLine(1), "trailing empty line")
}

func TestCheckEmptyComment(t *testing.T) {
	var n int
	pass := &analysis.Pass{Report: func(analysis.Diagnostic) { n++ }}
	lines := []commentLine{cl(" text,"), cl(""), cl(" more.")}
	checkEmptyComment(pass, cb(docNone, lines...), 1, nil)
	assert.Equal(t, 1, n)
	checkEmptyComment(pass, cb(docFunc, lines...), 1, nil)
	assert.Equal(t, 1, n, "blank line in doc comment exempt")
}

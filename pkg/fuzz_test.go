package easyreader

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.org/x/tools/go/analysis"
)

// fuzzSeeds cover fixture-style sources and edge cases: unicode, long lines, block comments, malformed Go.
var fuzzSeeds = []string{
	"package p\n",
	"// doc.\npackage p\n",
	"// line one\n// line two.\nfunc f() {}\n",
	"/* block\nc comment. */\nfunc f() {}\n",
	"func f() {\n\tx := 1 // trailing.\n\t_ = x\n}\n",
	"//go:generate foo\npackage p\n",
	"// \u2013 dash \u00a0 nbsp.\npackage p\n",
	"// " + strings.Repeat("word ", 60) + "\npackage p\n",
	"package p\n\n// no punct\nfunc f() {}\n",
	"!!!!not go",
	"\n\n// only comment.\n",
}

// fuzzRun is the result of running the analyzer over one in-memory source.
type fuzzRun struct {
	fset  *token.FileSet
	file  *ast.File
	diags []analysis.Diagnostic
	fixes []analysis.SuggestedFix
}

// analyzeSrc parses src and runs the analyzer fully in memory; no real files are read.
// It returns nil for unparseable input.
func analyzeSrc(t testing.TB, src []byte) *fuzzRun {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fuzz.go", src, parser.ParseComments)
	if err != nil {
		return nil
	}
	res := &fuzzRun{
		fset: fset,
		file: file,
	}
	pass := &analysis.Pass{
		Fset:  fset,
		Files: []*ast.File{file},
		Report: func(d analysis.Diagnostic) {
			assertDiagBounds(t, fset.File(file.Pos()), d)
			res.diags = append(res.diags, d)
			res.fixes = append(res.fixes, d.SuggestedFixes...)
		},
		ReadFile: func(string) ([]byte, error) { return src, nil },
	}
	s := DefaultSettings()
	_, err = run(&s)(pass)
	require.NoError(t, err)
	return res
}

// assertDiagBounds checks that a diagnostic and its fix edits stay inside the file.
func assertDiagBounds(t testing.TB, tf *token.File, d analysis.Diagnostic) {
	base, size := token.Pos(tf.Base()), token.Pos(tf.Size())
	in := func(p token.Pos) bool { return p.IsValid() && p >= base && p <= base+size }
	assert.True(t, in(d.Pos), "diagnostic pos out of bounds: %v", d.Pos)
	if d.End.IsValid() {
		assert.True(t, in(d.End) && d.Pos <= d.End, "diagnostic end out of bounds: %v", d.End)
	}
	for _, fx := range d.SuggestedFixes {
		last := base
		for _, e := range fx.TextEdits {
			assert.True(t, in(e.Pos) && in(e.End) && e.Pos <= e.End, "fix edit out of bounds: %v-%v", e.Pos, e.End)
			assert.True(t, e.Pos >= last, "fix edits overlap or are unsorted")
			last = e.End
		}
	}
}

// applyFixes dedupes identical fixes and applies their edits left to right,
// skipping overlaps as golangci-lint does. It returns nil when nothing applies.
func applyFixes(src []byte, tf *token.File, fixes []analysis.SuggestedFix) []byte {
	seen := map[string]bool{}
	var edits []analysis.TextEdit
	for _, fx := range fixes {
		key := fmt.Sprintf("%v", fx.TextEdits)
		if seen[key] {
			continue
		}
		seen[key] = true
		edits = append(edits, fx.TextEdits...)
	}
	if len(edits) == 0 {
		return nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].Pos < edits[j].Pos })
	base, done := tf.Base(), 0
	var out []byte
	for _, e := range edits {
		start, end := int(e.Pos)-base, int(e.End)-base
		if start < 0 || end < start || end > len(src) || start < done {
			continue
		}
		out = append(out, src[done:start]...)
		out = append(out, e.NewText...)
		done = end
	}
	return append(out, src[done:]...)
}

// FuzzAnalyzer runs the analyzer on arbitrary sources, asserting no panics,
// and that every diagnostic and fix edit stays inside the file.
func FuzzAnalyzer(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		analyzeSrc(t, src)
	})
}

// FuzzFixpoint applies the suggested fixes, then re-runs the analyzer.
// The fixed source must process safely, and empty-line diagnostics never grow.
func FuzzFixpoint(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		res := analyzeSrc(t, src)
		if res == nil || len(res.fixes) == 0 {
			return
		}
		fixed := applyFixes(src, res.fset.File(res.file.Pos()), res.fixes)
		if fixed == nil {
			return
		}
		after := analyzeSrc(t, fixed)
		if after == nil {
			return
		}
		// Empty-line is the only kind whose fixes are guaranteed monotonic today;
		// width, block-length and punctuation can grow (see the long-lines TODO).
		assert.LessOrEqual(t, countKind(after.diags, "empty comment line"), countKind(res.diags, "empty comment line"),
			"fixes increased empty-line diagnostics")
	})
}

// countKind counts diagnostics whose message carries the given prefix.
func countKind(diags []analysis.Diagnostic, prefix string) int {
	n := 0
	for _, d := range diags {
		if strings.HasPrefix(d.Message, prefix) {
			n++
		}
	}
	return n
}

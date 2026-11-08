// Package easyreader implements a go/analysis linter for comment style:
// line width, punctuation, block length, strange characters, blank lines.
package easyreader

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the easyreader analyzer with DefaultSettings.
var Analyzer = NewAnalyzer(DefaultSettings())

// NewAnalyzer returns an analyzer configured by s (defaults applied to unset fields).
// Settings are also exposed as analyzer flags for vet-tool and singlechecker runs.
func NewAnalyzer(s Settings) *analysis.Analyzer {
	settings := s.WithDefaults()
	a := &analysis.Analyzer{
		Name: "easyreader",
		Doc:  "reports comment style violations: line width, terminal punctuation, block length, ambiguous unicode, interior blank lines",
		Run:  run(&settings),
	}
	a.Flags.IntVar(
		&settings.MaxLineWidth,
		"max-line-width",
		settings.MaxLineWidth,
		"maximum comment width in characters",
	)
	a.Flags.IntVar(
		&settings.MaxBlockLines,
		"max-block-lines",
		settings.MaxBlockLines,
		"maximum inline comment block lines on exported declarations (negative disables the check)",
	)
	a.Flags.IntVar(
		&settings.MaxBlockLinesPrivate,
		"max-block-lines-private",
		settings.MaxBlockLinesPrivate,
		"maximum inline comment block lines on unexported declarations, including their bodies (inherits max-block-lines; negative disables the check)",
	)
	a.Flags.StringVar(
		&settings.EndPunctuation,
		"end-punctuation",
		settings.EndPunctuation,
		"allowed terminal punctuation characters (the first is appended by the auto-fix)",
	)
	a.Flags.BoolVar(
		&settings.CheckGenerated,
		"check-generated",
		settings.CheckGenerated,
		"lint files carrying the Code generated marker",
	)
	a.Flags.Var(
		(*csvFlag)(&settings.ExemptionMarkers),
		"exemption-markers",
		"comma-separated block-length exemption substrings",
	)
	return a
}

// Main run step called by the Analyzer.
func run(s *Settings) func(*analysis.Pass) (any, error) {
	return func(pass *analysis.Pass) (any, error) {
		cfg := *s // snapshot after flag parsing.
		for _, f := range pass.Files {
			if !cfg.CheckGenerated && ast.IsGenerated(f) {
				continue
			}
			checkFile(pass, f, cfg)
		}
		return nil, nil
	}
}

// validates every comment group in given file.
// Comment groups expand to per-line records; standalone blocks are checked for block-length.
func checkFile(pass *analysis.Pass, file *ast.File, s Settings) {
	metas := commentMetas(file)
	src := newFileSource(pass, file)

	firstBlock := true // the first standalone non-doc comment block in a file is exempt.
	for _, g := range file.Comments {
		lines := commentLines(pass, src, g)
		meta := metas[g]

		for _, l := range lines {
			checkStrangeChars(pass, l)
			if l.trailing {
				checkLineWidth(pass, s.MaxLineWidth, l, nil)
			}
		}

		// Split into blocks of standalone lines; a trailing line ends a block.
		for _, block := range commentBlocks(lines, meta) {
			fix := reflowFix(s, block)
			for i, l := range block.lines {
				checkLineWidth(pass, s.MaxLineWidth, l, fix)
				checkPunctuation(pass, s, l)
				checkEmptyComment(pass, block, i, fix)
			}
			first := firstBlock && !block.meta.isDoc()
			if first {
				firstBlock = false
			}
			checkBlockLength(pass, s.blockLimit(block.meta), block, block.isExempt(s, first), fix)
		}
	}
}

// commentBlock is a run of standalone comment lines plus its group's linting metadata.
type commentBlock struct {
	lines []commentLine
	meta  commentMeta
}

// isDoc reports whether the block is a doc comment (paragraph breaks allowed).
func (b commentBlock) isDoc() bool { return b.meta.isDoc() }

// isSlashBlock reports whether every line in the block is a // comment (no /* */).
func (b commentBlock) isSlashBlock() bool {
	for _, l := range b.lines {
		if !l.slash {
			return false
		}
	}
	return true
}

// commentBlocks splits lines into blocks of non-trailing comment lines, stamped with meta.
// A trailing comment has code before it on the same line; it ends the current block.
func commentBlocks(lines []commentLine, meta commentMeta) []commentBlock {
	var out []commentBlock
	var cur []commentLine
	flush := func() {
		if len(cur) > 0 {
			out = append(out, commentBlock{lines: cur, meta: meta})
			cur = nil
		}
	}
	for _, l := range lines {
		if l.trailing {
			flush()
			continue
		}
		cur = append(cur, l)
	}
	flush()
	return out
}

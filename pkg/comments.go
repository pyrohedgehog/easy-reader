package easyreader

import (
	"bytes"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// commentLine is one physical source line of comment text.
// Markers are stripped; these records are the unit the rules operate on.
type commentLine struct {
	pos      token.Pos // start of the comment on this line
	end      token.Pos // end of the comment text on this line
	fixPos   token.Pos // where a punctuation fix inserts; NoPos for /* */ lines
	text     string    // comment content with // or /* */ markers stripped
	indent   string    // leading whitespace before the comment on its line
	slash    bool      // line comes from a // comment (vs /* */)
	trailing bool      // code precedes the comment on this line
}

// fileSource lazily loads raw source lines, used to detect whether code precedes a comment on its line.
type fileSource struct {
	lines [][]byte
}

func newFileSource(pass *analysis.Pass, f *ast.File) *fileSource {
	tf := pass.Fset.File(f.Pos())
	if tf == nil {
		return nil
	}
	read := pass.ReadFile
	if read == nil {
		read = os.ReadFile // analysistest does not provide ReadFile
	}
	b, err := read(tf.Name())
	if err != nil {
		return nil
	}
	return &fileSource{lines: bytes.Split(b, []byte("\n"))}
}

func (fs *fileSource) hasCodeBefore(p token.Position) bool {
	if fs == nil || p.Line < 1 || p.Line > len(fs.lines) {
		return false
	}
	col := p.Column - 1
	if col < 0 || col > len(fs.lines[p.Line-1]) {
		return false
	}
	return len(bytes.TrimSpace(fs.lines[p.Line-1][:col])) > 0
}

// lineIndent returns the leading whitespace of p's source line for reflow fix indentation.
// Falls back to column-derived spaces when source is unavailable.
func (fs *fileSource) lineIndent(p token.Position) string {
	if fs == nil || p.Line < 1 || p.Line > len(fs.lines) {
		return strings.Repeat(" ", p.Column-1)
	}
	raw := fs.lines[p.Line-1]
	i := 0
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t') {
		i++
	}
	return string(raw[:i])
}

// commentLines expands a comment group into per-line records.
// A `//` comment is one line; a `/* */` comment contributes one record per physical line it spans.
func commentLines(pass *analysis.Pass, fs *fileSource, g *ast.CommentGroup) []commentLine {
	var out []commentLine
	for i, c := range g.List {
		start := pass.Fset.Position(c.Slash)
		trailing := fs.hasCodeBefore(start)

		if strings.HasPrefix(c.Text, "//") {
			text := c.Text[2:]
			out = append(out, commentLine{
				pos: c.Slash, end: c.End(),
				fixPos:   c.Slash + token.Pos(2+len(strings.TrimRight(text, " \t"))),
				text:     text,
				indent:   fs.lineIndent(start),
				slash:    true,
				trailing: trailing,
			})
			continue
		}

		// A /* */ block starting mid-line is an inline annotation,
		// so its lines are marked trailing: length-checked only.
		if i == 0 && trailing {
			out = append(out, commentLine{
				pos: c.Slash, end: c.End(),
				text: c.Text, trailing: true,
			})
			continue
		}

		out = append(out, splitBlockComment(pass, fs, c)...)
	}
	return out
}

// splitBlockComment expands a multi-line /* */ comment,
// stripping the delimiters and leading `*` from each line.
func splitBlockComment(pass *analysis.Pass, fs *fileSource, c *ast.Comment) []commentLine {
	parts := strings.Split(c.Text, "\n")
	out := make([]commentLine, 0, len(parts))

	off := 0
	for i, part := range parts {
		pos := c.Slash + token.Pos(off)
		off += len(part) + 1 // +1 for the split newline

		text := part
		if i == 0 {
			text = strings.TrimPrefix(text, "/*")
		}
		if i == len(parts)-1 {
			if j := strings.Index(text, "*/"); j >= 0 {
				text = text[:j]
			}
		} else if t := strings.TrimLeft(text, " \t"); strings.HasPrefix(t, "*") {
			text = t[1:]
		}

		lineStart := pass.Fset.Position(pos)
		out = append(out, commentLine{
			pos: pos, end: pos + token.Pos(len(text)),
			text: text, trailing: fs.hasCodeBefore(lineStart),
		})
	}
	return out
}

// docKind identifies what declaration a doc comment documents; docNone for non-doc comments.
type docKind int

const (
	docNone docKind = iota
	docPackage
	docFunc
	docType
	docField
	docValue
	docImport
)

// commentMeta is the linting metadata of a comment group.
// exported marks public declarations; doc marks doc comments (paragraph breaks allowed).
type commentMeta struct {
	doc      docKind
	exported bool
}

// isDoc reports whether the group is a doc comment.
func (m commentMeta) isDoc() bool { return m.doc != docNone }

// commentMetas returns per-group linting metadata.
// Doc comments get their declaration's kind and exportedness; others inherit the enclosing declaration's.
func commentMetas(f *ast.File) map[*ast.CommentGroup]commentMeta {
	metas := map[*ast.CommentGroup]commentMeta{}
	add := func(g *ast.CommentGroup, doc docKind, exported bool) {
		if g != nil {
			metas[g] = commentMeta{doc: doc, exported: exported}
		}
	}
	add(f.Doc, docPackage, true)
	ast.Inspect(f, func(n ast.Node) bool {
		// TODO: make clearer.
		switch n := n.(type) {
		case *ast.FuncDecl:
			add(n.Doc, docFunc, n.Name.IsExported())
		case *ast.GenDecl:
			add(n.Doc, genDeclDocKind(n), specsExported(n.Specs))
		case *ast.TypeSpec:
			add(n.Doc, docType, n.Name.IsExported())
		case *ast.ValueSpec:
			add(n.Doc, docValue, namesExported(n.Names))
		case *ast.ImportSpec:
			add(n.Doc, docImport, false)
		case *ast.Field:
			exported := namesExported(n.Names)
			if len(n.Names) == 0 {
				exported = embeddedExported(n.Type)
			}
			add(n.Doc, docField, exported)
		}
		return true
	})
	// Non-doc comments inherit the exportedness of their enclosing declaration.
	// TODO: make this inheritance configurable.
	for _, g := range f.Comments {
		if _, ok := metas[g]; !ok {
			metas[g] = commentMeta{doc: docNone, exported: enclosingDeclExported(f, g)}
		}
	}
	return metas
}

// genDeclDocKind maps a GenDecl's token to the doc kind of its doc comment.
func genDeclDocKind(d *ast.GenDecl) docKind {
	switch d.Tok {
	case token.TYPE:
		return docType
	case token.IMPORT:
		return docImport
	default:
		return docValue
	}
}

// enclosingDeclExported reports whether g falls inside the body of an exported top-level declaration.
func enclosingDeclExported(f *ast.File, g *ast.CommentGroup) bool {
	pos := g.Pos()
	for _, decl := range f.Decls {
		if pos < decl.Pos() || pos > decl.End() {
			continue
		}
		switch d := decl.(type) {
		case *ast.FuncDecl:
			return d.Name.IsExported()
		case *ast.GenDecl:
			return specsExported(d.Specs)
		}
	}
	return false
}

// TODO: see if an interface simplifies these.
func namesExported(names []*ast.Ident) bool {
	for _, n := range names {
		if n.IsExported() {
			return true
		}
	}
	return false
}

func specsExported(specs []ast.Spec) bool {
	for _, spec := range specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if s.Name.IsExported() {
				return true
			}
		case *ast.ValueSpec:
			if namesExported(s.Names) {
				return true
			}
		}
	}
	return false
}

// embeddedExported reports whether an embedded field's type is exported:
// T, *T, or pkg.T with an exported T.
func embeddedExported(e ast.Expr) bool {
	switch t := e.(type) {
	case *ast.StarExpr:
		return embeddedExported(t.X)
	case *ast.Ident:
		return t.IsExported()
	case *ast.SelectorExpr:
		return t.Sel.IsExported()
	}
	return false
}

// isExempt reports whether the block is exempt from block-length:
// it is the file's first standalone non-doc block, or a line carries an exemption marker.
func (b commentBlock) isExempt(s Settings, firstBlock bool) bool {
	if firstBlock {
		return true
	}
	for _, line := range b.lines {
		if lineHasMarker(line.text, s) {
			return true
		}
	}
	return false
}

package easyreader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis"
)

// TestPunctLongTail covers the punctuation exemption for slash-joined last words.
func TestPunctLongTail(t *testing.T) {
	s := DefaultSettings()
	tests := []struct {
		name      string
		text      string
		wantDiags int
	}{
		{"URL tail exempt", " see https://example.com/a/very/long/url/segment/that/keeps/going", 0},
		{"path tail exempt", " see /usr/local/lib/very/long/path/segment/that/keeps/going/on", 0},
		{"plain line flagged", " see the docs for details", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n int
			pass := &analysis.Pass{Report: func(analysis.Diagnostic) { n++ }}
			checkPunctuation(pass, s, cl(tt.text))
			assert.Equal(t, tt.wantDiags, n)
		})
	}
}

// TestIsDirective covers compiler/toolchain directives that cannot live in analysistest fixtures:
// //line remaps positions and breaks expectation matching, so it is asserted here.
func TestIsDirective(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"go:generate", "go:generate goyacc", true},
		{"+build", "+build linux", true},
		{"line directive", "line foo.go:10:5", true},
		{"export directive", "export F", true},
		{"nolint", "nolint:foo", true},
		{"lint", "lint:ignore", true},
		{"bare line", "line", true},
		{"bare export", "export", true},
		{"normal comment", "this is a normal comment", false},
		{"exported prefix", "exported is true for doc comments", false},
		{"lineup prefix", "lineup schedule", false},
		{"lineman prefix", "lineman on duty", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isDirective(tt.in))
		})
	}
}

// TestLineWidthLongTail covers the exemption for lines whose excess is a slash-joined last word.
func TestLineWidthLongTail(t *testing.T) {
	head := strings.Repeat("w", 140)
	tests := []struct {
		name      string
		text      string
		wantDiags int
	}{
		{"long URL tail exempt", " " + head + " https://example.com/a/very/long/url/segment/that/keeps/going", 0},
		{"long path tail exempt", " " + head + " /usr/local/lib/very/long/path/segment/that/keeps/going/on", 0},
		{"plain overlong line flagged", " " + head + " " + strings.Repeat("w", 20), 1},
		{"overlong prose still flagged despite URL tail", " " + strings.Repeat("w", 160) + " https://example.com/x", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n int
			pass := &analysis.Pass{Report: func(analysis.Diagnostic) { n++ }}
			checkLineWidth(pass, 150, cl(tt.text), nil)
			assert.Equal(t, tt.wantDiags, n)
		})
	}
}

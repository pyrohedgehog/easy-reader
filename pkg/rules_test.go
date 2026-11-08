package easyreader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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

package easyreader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCsvFlag(t *testing.T) {
	var c csvFlag
	require.NoError(t, c.Set(""))
	assert.Empty(t, c, "empty marker would exempt every run")
	require.NoError(t, c.Set("a, b,,c"))
	assert.Equal(t, "a,b,c", c.String())
}

func TestBlockLimitCascade(t *testing.T) {
	s := Settings{
		MaxBlockLines:        2,
		MaxBlockLinesPrivate: 2,
		PackageDoc: BlockLimits{
			MaxBlockLines: -1,
		},
		TypeDoc: BlockLimits{
			MaxBlockLines:        9,
			MaxBlockLinesPrivate: 7,
		},
		FuncDoc: BlockLimits{MaxBlockLines: 5},
	}
	tests := []struct {
		name string
		meta commentMeta
		want int
	}{
		{"exported type doc", commentMeta{doc: docType, exported: true}, 9},
		{"unexported type doc", commentMeta{doc: docType, exported: false}, 7},
		{"exported func doc", commentMeta{doc: docFunc, exported: true}, 5},
		{"package doc negative disables the check", commentMeta{doc: docPackage, exported: true}, -1},
		{"unexported func doc falls to kind public", commentMeta{doc: docFunc, exported: false}, 5},
		{"exported value doc falls to global", commentMeta{doc: docValue, exported: true}, 2},
		{"unexported non-doc falls to global private", commentMeta{doc: docNone, exported: false}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, s.blockLimit(tt.meta))
		})
	}
}

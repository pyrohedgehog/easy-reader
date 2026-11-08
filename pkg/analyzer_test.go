package easyreader

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	// Package a includes generated.go, skipped by default,
	// so diagnostics from it would be unexpected and fail this test.
	analysistest.Run(t, analysistest.TestData(), Analyzer, "a")
}

func TestCheckGenerated(t *testing.T) {
	s := DefaultSettings()
	s.CheckGenerated = true
	analysistest.Run(t, analysistest.TestData(), NewAnalyzer(s), "gencheck")
}

func TestPrivateBlockLines(t *testing.T) {
	s := DefaultSettings()
	s.MaxBlockLinesPrivate = 1
	analysistest.Run(t, analysistest.TestData(), NewAnalyzer(s), "private")
}

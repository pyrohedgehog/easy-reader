// Package plugin adapts easyreader to golangci-lint's module plugin contract.
// Build a custom binary in the consuming repo; see the README for setup.
package plugin

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	easyreader "github.com/pyrohedgehog/easy-reader/pkg"
)

func init() {
	register.Plugin("easyreader", New)
}

type plugin struct {
	settings easyreader.Settings
}

// New is the plugin factory; it receives the .golangci.yml settings map.
func New(conf any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[easyreader.Settings](conf)
	if err != nil {
		return nil, err
	}
	return plugin{settings: s.WithDefaults()}, nil
}

func (p plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{easyreader.NewAnalyzer(p.settings)}, nil
}

// GetLoadMode returns LoadModeSyntax: comments need only parsed files,
// not type information.
func (p plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

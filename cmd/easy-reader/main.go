// easy-reader is a go vet tool exposing the easyreader analyzer; usage is in the README.
// Note: nolint directives are honored only under golangci-lint; under go vet, use exemption markers.
package main

import (
	"golang.org/x/tools/go/analysis/unitchecker"

	easyreader "github.com/pyrohedgehog/easy-reader/pkg"
)

func main() {
	unitchecker.Main(easyreader.Analyzer)
}

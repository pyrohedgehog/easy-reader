// Package private is a fixture for max-block-lines-private.
package private

// A standalone comment that consumes the first-run exemption.

func Exported() {
	// Inline block inside an exported function.
	// Uses the public max-block-lines limit.
	_ = 1
}

func unexported() {
	// Inline block inside an unexported function. // want "comment block is 2 lines" "comment does not end"
	// Uses the private max-block-lines-private limit. // want "comment does not end"
	_ = 1
}

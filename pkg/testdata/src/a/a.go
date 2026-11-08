// Package a is a test fixture. // want "comment does not end"
// Its two-line package doc sits exactly at the block-length limit.
package a

// A compliant comment.
func Good() {}

// Missing terminal punctuation // want "comment does not end"
func Punct() {}

var x = 1 // trailing comments skip the punctuation check

// Doc comments share the public block-length limit, // want "comment block is 3 lines" "comment does not end"
// so exported declarations allow two lines, // want "comment does not end"
// and this third line is over.
func Documented() {}

func Three() {
	x := 1
	// block run exceeding max: // want "comment does not end"
	// middle line. // want "comment does not end"
	// last line.
	_ = x
}

func Four() {
	x := 1
	// Another three-line block, later in the file: // want "comment block is 3 lines" "comment does not end"
	// it is no longer the first run, so the rule // want "comment does not end"
	// applies to it.
	_ = x
}

// Long but exempt via marker: comment-length-exception.
// more. // want "comment does not end"
// more. // want "comment does not end"
// more.
func Exempt() {}

// Contains an en dash–inside // want "comment contains en dash" "comment does not end"
func Dash() {}

// Directive lines are exempt from punctuation and length.
//
//go:generate something
//nolint:whatever
//lint:ignore
//export Directives
func Directives() {}

// This standalone comment line is deliberately much longer than one hundred and fifty characters so that the line-width rule reports it here in the analysistest fixture. // want "comment exceeds 150 width" "comment does not end"
func Long() {}

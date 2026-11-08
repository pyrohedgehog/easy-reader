package a

// FirstRunExemption tests the first standalone non-doc comment run exemption.
// This file is separate from a.go so it gets its own firstBlock counter.
func FirstRunExemption() {
	// This three-line inline block is the first standalone // want "comment does not end"
	// non-doc comment run in this file, so the block-length // want "comment does not end"
	// rule exempts it.
	_ = 1
}

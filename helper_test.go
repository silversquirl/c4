// Helpers for tests, not tests for helpers
package main

func spc(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func CodeCompare(a, b string) (eq bool, ai, bi int) {
	// Compare without taking into account indentation
	for {
		aspc := false
		for ai < len(a) && spc(a[ai]) {
			ai++
			aspc = true
		}
		bspc := false
		for bi < len(b) && spc(b[bi]) {
			bi++
			bspc = true
		}

		end0 := ai >= len(a)
		end1 := bi >= len(b)
		if end0 && end1 {
			// Strings are equal
			return true, -1, -1
		} else if end0 || end1 {
			// Strings are different lengths
			return
		}

		if ai == 0 {
			aspc = true
		}
		if bi == 0 {
			bspc = true
		}

		if aspc != bspc || a[ai] != b[bi] {
			// Strings differ
			return
		}

		ai++
		bi++
	}
}

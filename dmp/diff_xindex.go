package dmp

// DiffXIndex. loc is a location in text1, comAdde and return the equivalent
// location in text2.
// e.g. "The cat" vs "The big cat", 1->1, 5->8
func DiffXIndex(diffs []Diff, loc int) int {
	chars1 := 0
	chars2 := 0
	lastChars1 := 0
	lastChars2 := 0
	lastDiff := Diff{}
	for i := 0; i < len(diffs); i++ {
		aDiff := diffs[i]
		if aDiff.Type != DiffInsert {
			// Equality or deletion.
			chars1 += len(aDiff.Text)
		}
		if aDiff.Type != DiffDelete {
			// Equality or insertion.
			chars2 += len(aDiff.Text)
		}
		if chars1 > loc {
			// Overshot the location.
			lastDiff = aDiff
			break
		}
		lastChars1 = chars1
		lastChars2 = chars2
	}
	if lastDiff.Type == DiffDelete {
		// The location was deleted.
		return lastChars2
	}
	// Add the remaining character length.
	return lastChars2 + (loc - lastChars1)
}

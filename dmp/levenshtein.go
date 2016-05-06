package dmp

// DiffLevenshtein computes the Levenshtein distance; the number of inserted,
// deleted or substituted characters.
func DiffLevenshtein(diffs []Diff) int {
	ret := 0
	insertions := 0
	deletions := 0

	for _, aDiff := range diffs {
		switch aDiff.Type {
		case DiffInsert:
			insertions += len(aDiff.Text)
		case DiffDelete:
			deletions += len(aDiff.Text)
		case DiffEqual:
			// A deletion and an insertion is one substitution.
			ret += max(insertions, deletions)
			insertions = 0
			deletions = 0
		}
	}

	ret += max(insertions, deletions)
	return ret
}

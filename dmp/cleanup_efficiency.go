package dmp

func diffCleanupEfficiency(diffs []Diff, editCost int) []Diff {
	changes := false
	// Stack of indices where equalities are found.
	equalities := new(Stack)
	// Always equal to equalities[equalitiesLength-1][1]
	lastequality := ""
	i := 0 // Index of current position.
	// Is there an insertion operation before the last equality.
	preIns := false
	// Is there a deletion operation before the last equality.
	preDel := false
	// Is there an insertion operation after the last equality.
	postIns := false
	// Is there a deletion operation after the last equality.
	postDel := false
	for i < len(diffs) {
		if diffs[i].Type == DiffEqual { // Equality found.
			if len(diffs[i].Text) < editCost &&
				(postIns || postDel) {
				// Candidate found.
				equalities.Push(i)
				preIns = postIns
				preDel = postDel
				lastequality = diffs[i].Text
			} else {
				// Not a candidate, and can never become one.
				equalities.Clear()
				lastequality = ""
			}
			postIns = false
			postDel = false
		} else { // An insertion or deletion.
			if diffs[i].Type == DiffDelete {
				postDel = true
			} else {
				postIns = true
			}
			/*
			 * Five types to be split:
			 * <ins>A</ins><del>B</del>XY<ins>C</ins><del>D</del>
			 * <ins>A</ins>X<ins>C</ins><del>D</del>
			 * <ins>A</ins><del>B</del>X<ins>C</ins>
			 * <ins>A</del>X<ins>C</ins><del>D</del>
			 * <ins>A</ins><del>B</del>X<del>C</del>
			 */
			var sum_pres int
			if preIns {
				sum_pres++
			}
			if preDel {
				sum_pres++
			}
			if postIns {
				sum_pres++
			}
			if postDel {
				sum_pres++
			}
			if len(lastequality) > 0 &&
				((preIns && preDel && postIns && postDel) ||
					((len(lastequality) < editCost/2) &&
						sum_pres == 3)) {

				// Duplicate record.
				diffs = append(
					diffs[:equalities.Peek().(int)],
					append(
						[]Diff{{DiffDelete, lastequality}},
						diffs[equalities.Peek().(int):]...,
					)...,
				)

				// Change second copy to insert.
				diffs[equalities.Peek().(int)+1].Type = DiffInsert
				equalities.Pop() // Throw away the equality we just deleted.
				lastequality = ""

				if preIns && preDel {
					// No changes made which could affect previous entry, keep
					// going.
					postIns = true
					postDel = true
					equalities.Clear()
				} else {
					if equalities.Len() > 0 {
						equalities.Pop()
						i = equalities.Peek().(int)
					} else {
						i = -1
					}
					postIns = false
					postDel = false
				}
				changes = true
			}
		}
		i++
	}

	if changes {
		diffs = DiffCleanupMerge(diffs)
	}

	return diffs
}

package dmp

import (
	"math"
	"time"
)

// The data structure representing a diff is an array of tuples:
// [[DiffDelete, 'Hello'], [DiffInsert, 'Goodbye'], [DiffEqual, ' world.']]
// which means: delete 'Hello', add 'Goodbye' and keep ' world.'

// DiffMain finds the differences between two texts.
func (dmp *DMP) DiffMain(s1, s2 string, checkLines bool) []Diff {
	return dmp.diffMain(s1, s2, checkLines, deadline(dmp.DiffTimeout))
}

func (dmp *DMP) diffMain(
	s1, s2 string, checkLines bool, deadline time.Time,
) []Diff {
	return dmp.diffMainRunes([]rune(s1), []rune(s2), checkLines, deadline)
}

// DiffMainRunes finds the differences between two rune sequences.
func (dmp *DMP) DiffMainRunes(s1, s2 []rune, checkLines bool) []Diff {
	return dmp.diffMainRunes(s1, s2, checkLines, deadline(dmp.DiffTimeout))
}

func (dmp *DMP) diffMainRunes(
	s1, s2 []rune, checkLines bool, deadline time.Time,
) []Diff {
	if runesEqual(s1, s2) {
		var diffs []Diff
		if len(s1) > 0 {
			diffs = append(diffs, Diff{DiffEqual, string(s1)})
		}
		return diffs
	}
	// Trim off common prefix (speedup).
	n := commonPrefixLength(s1, s2)
	prefix := s1[:n]
	s1 = s1[n:]
	s2 = s2[n:]

	// Trim off common suffix (speedup).
	n = commonSuffixLength(s1, s2)
	suffix := s1[len(s1)-n:]
	s1 = s1[:len(s1)-n]
	s2 = s2[:len(s2)-n]

	// Compute the diff on the middle block.
	diffs := dmp.diffCompute(s1, s2, checkLines, deadline)

	// Restore the prefix and suffix.
	if len(prefix) != 0 {
		diffs = diffPrepend(diffEq(string(prefix)), diffs)
	}
	if len(suffix) != 0 {
		diffs = diffAppend(diffs, diffEq(string(suffix)))
	}
	return DiffCleanupMerge(diffs)
}

// diffCompute finds the differences between two rune slices.  Assumes that
// the texts do not have any common prefix or suffix.
func (dmp *DMP) diffCompute(
	text1, text2 []rune, checkLines bool, deadline time.Time,
) []Diff {
	diffs := []Diff{}
	if len(text1) == 0 {
		// Just add some text (speedup).
		return append(diffs, Diff{DiffInsert, string(text2)})
	} else if len(text2) == 0 {
		// Just delete some text (speedup).
		return append(diffs, Diff{DiffDelete, string(text1)})
	}

	var longtext, shorttext []rune
	if len(text1) > len(text2) {
		longtext = text1
		shorttext = text2
	} else {
		longtext = text2
		shorttext = text1
	}

	if i := runesIndex(longtext, shorttext); i != -1 {
		op := DiffInsert
		// Swap insertions for deletions if diff is reversed.
		if len(text1) > len(text2) {
			op = DiffDelete
		}
		// Shorter text is inside the longer text (speedup).
		return []Diff{
			{op, string(longtext[:i])},
			{DiffEqual, string(shorttext)},
			{op, string(longtext[i+len(shorttext):])},
		}
	} else if len(shorttext) == 1 {
		// Single character string.
		// After the previous speedup, the character can't be an equality.
		return []Diff{
			{DiffDelete, string(text1)},
			{DiffInsert, string(text2)},
		}
		// Check to see if the problem can be split in two.
	} else if hm := diffHalfMatch(dmp, text1, text2); hm != nil {
		// A half-match was found, sort out the return data.
		text1_a := hm[0]
		text1_b := hm[1]
		text2_a := hm[2]
		text2_b := hm[3]
		mid_common := hm[4]
		// Send both pairs off for separate processing.
		diffs_a := dmp.diffMainRunes(text1_a, text2_a, checkLines, deadline)
		diffs_b := dmp.diffMainRunes(text1_b, text2_b, checkLines, deadline)
		// Merge the results.
		return append(diffs_a, append(
			[]Diff{{DiffEqual, string(mid_common)}}, diffs_b...,
		)...)
	} else if checkLines && len(text1) > 100 && len(text2) > 100 {
		return dmp.diffLineMode(text1, text2, deadline)
	}
	return dmp.diffBisect(text1, text2, deadline)
}

// diffLineMode does a quick line-level diff on both []runes, then rediff the
// parts for greater accuracy. This speedup can produce non-minimal diffs.
func (dmp *DMP) diffLineMode(text1, text2 []rune, deadline time.Time) []Diff {
	// Scan the text on a line-by-line basis first.
	text1, text2, linearray := diffLinesToRunes(text1, text2)

	diffs := dmp.diffMainRunes(text1, text2, false, deadline)

	// Convert the diff back to original text.
	diffs = DiffCharsToLines(diffs, linearray)
	// Eliminate freak matches (e.g. blank lines)
	diffs = DiffCleanupSemantic(diffs)

	// Rediff any replacement blocks, this time character-by-character.
	// Add a dummy entry at the end.
	diffs = append(diffs, Diff{DiffEqual, ""})

	pointer := 0
	count_delete := 0
	count_insert := 0
	text_delete := ""
	text_insert := ""

	for pointer < len(diffs) {
		switch diffs[pointer].Type {
		case DiffInsert:
			count_insert++
			text_insert += diffs[pointer].Text
		case DiffDelete:
			count_delete++
			text_delete += diffs[pointer].Text
		case DiffEqual:
			// Upon reaching an equality, check for prior redundancies.
			if count_delete >= 1 && count_insert >= 1 {
				// Delete the offending records and add the merged ones.
				diffs = splice(diffs, pointer-count_delete-count_insert,
					count_delete+count_insert)

				pointer = pointer - count_delete - count_insert
				a := dmp.diffMain(text_delete, text_insert, false, deadline)
				for j := len(a) - 1; j >= 0; j-- {
					diffs = splice(diffs, pointer, 0, a[j])
				}
				pointer = pointer + len(a)
			}

			count_insert = 0
			count_delete = 0
			text_delete = ""
			text_insert = ""
		}
		pointer++
	}

	return diffs[:len(diffs)-1] // Remove the dummy entry at the end.
}

// DiffBisect finds the 'middle snake' of a diff, split the problem in two
// and return the recursively constructed diff.
// See Myers 1986 paper: An O(ND) Difference Algorithm and Its Variations.
func (dmp *DMP) DiffBisect(s1, s2 string, deadline time.Time) []Diff {
	// Unused in this code, but retained for interface compatibility.
	return dmp.diffBisect([]rune(s1), []rune(s2), deadline)
}

// diffBisect finds the 'middle snake' of a diff, splits the problem in two
// and returns the recursively constructed diff.
// See Myers's 1986 paper: An O(ND) Difference Algorithm and Its Variations.
func (dmp *DMP) diffBisect(s1, s2 []rune, deadline time.Time) []Diff {
	// Cache the text lengths to prevent multiple calls.
	len1, len2 := len(s1), len(s2)

	dmax := (len1 + len2 + 1) / 2
	offset := dmax
	vlen := 2 * dmax

	v1 := make([]int, vlen)
	v2 := make([]int, vlen)
	for i := range v1 {
		v1[i] = -1
		v2[i] = -1
	}
	v1[offset+1] = 0
	v2[offset+1] = 0

	delta := len1 - len2
	// If the total number of characters is odd, then the front path will
	// collide with the reverse path.
	front := delta%2 != 0
	// Offsets for start and end of k loop.
	// Prevents mapping of space beyond the grid.
	k1start := 0
	k1end := 0
	k2start := 0
	k2end := 0
	for d := 0; d < dmax; d++ {
		// Bail out if deadline is reached.
		if time.Now().After(deadline) {
			break
		}

		// Walk the front path one step.
		for k1 := -d + k1start; k1 <= d-k1end; k1 += 2 {
			k1_offset := offset + k1
			var x1 int

			if k1 == -d || (k1 != d && v1[k1_offset-1] < v1[k1_offset+1]) {
				x1 = v1[k1_offset+1]
			} else {
				x1 = v1[k1_offset-1] + 1
			}

			y1 := x1 - k1
			for x1 < len1 && y1 < len2 {
				if s1[x1] != s2[y1] {
					break
				}
				x1++
				y1++
			}
			v1[k1_offset] = x1
			if x1 > len1 {
				// Ran off the right of the graph.
				k1end += 2
			} else if y1 > len2 {
				// Ran off the bottom of the graph.
				k1start += 2
			} else if front {
				k2_offset := offset + delta - k1
				if k2_offset >= 0 && k2_offset < vlen &&
					v2[k2_offset] != -1 {
					// Mirror x2 onto top-left coordinate system.
					x2 := len1 - v2[k2_offset]
					if x1 >= x2 {
						// Overlap detected.
						return dmp.diffBisectSplit(
							s1, s2, x1, y1, deadline,
						)
					}
				}
			}
		}
		// Walk the reverse path one step.
		for k2 := -d + k2start; k2 <= d-k2end; k2 += 2 {
			k2_offset := offset + k2
			var x2 int
			if k2 == -d || (k2 != d && v2[k2_offset-1] < v2[k2_offset+1]) {
				x2 = v2[k2_offset+1]
			} else {
				x2 = v2[k2_offset-1] + 1
			}
			var y2 = x2 - k2
			for x2 < len1 && y2 < len2 {
				if s1[len1-x2-1] != s2[len2-y2-1] {
					break
				}
				x2++
				y2++
			}
			v2[k2_offset] = x2
			if x2 > len1 {
				// Ran off the left of the graph.
				k2end += 2
			} else if y2 > len2 {
				// Ran off the top of the graph.
				k2start += 2
			} else if !front {
				k1_offset := offset + delta - k2
				if k1_offset >= 0 && k1_offset < vlen &&
					v1[k1_offset] != -1 {
					x1 := v1[k1_offset]
					y1 := offset + x1 - k1_offset
					// Mirror x2 onto top-left coordinate system.
					x2 = len1 - x2
					if x1 >= x2 {
						// Overlap detected.
						return dmp.diffBisectSplit(
							s1, s2, x1, y1, deadline,
						)
					}
				}
			}
		}
	}
	// Diff took too long and hit the deadline or
	// number of diffs equals number of characters, no commonality at all.
	return []Diff{
		{DiffDelete, string(s1)},
		{DiffInsert, string(s2)},
	}
}

func (dmp *DMP) diffBisectSplit(runes1, runes2 []rune, x, y int,
	deadline time.Time) []Diff {
	runes1a := runes1[:x]
	runes2a := runes2[:y]
	runes1b := runes1[x:]
	runes2b := runes2[y:]

	// Compute both diffs serially.
	diffs := dmp.diffMainRunes(runes1a, runes2a, false, deadline)
	diffsb := dmp.diffMainRunes(runes1b, runes2b, false, deadline)

	return append(diffs, diffsb...)
}

// DiffHalfMatch checks whether the two texts share a substring which is at
// least half the length of the longer text. This speedup can produce
// non-minimal diffs.
func (dmp *DMP) DiffHalfMatch(text1, text2 string) []string {
	// Unused in this code, but retained for interface compatibility.
	rs := diffHalfMatch(dmp, []rune(text1), []rune(text2))
	if rs == nil {
		return nil
	}

	result := make([]string, len(rs))
	for i, r := range rs {
		result[i] = string(r)
	}
	return result
}

// DiffCleanupEfficiency reduces the number of edits by eliminating
// operationally trivial equalities.
func (dmp *DMP) DiffCleanupEfficiency(diffs []Diff) []Diff {
	return diffCleanupEfficiency(diffs, dmp.DiffEditCost)
}

//  MATCH FUNCTIONS

// MatchMain locates the best instance of 'pattern' in 'text' near 'loc'.
// Returns -1 if no match found.
func (dmp *DMP) MatchMain(s, pattern string, loc int) int {
	// Check for null inputs not needed since null can't be passed in C#.

	loc = int(math.Max(0, math.Min(float64(loc), float64(len(s)))))
	if s == pattern {
		// Shortcut (potentially not guaranteed by the algorithm)
		return 0
	} else if len(s) == 0 {
		// Nothing to match.
		return -1
	} else if loc+len(pattern) <= len(s) &&
		s[loc:loc+len(pattern)] == pattern {
		// Perfect match at the perfect spot!  (Includes case of null pattern)
		return loc
	}
	// Do a fuzzy compare.
	return dmp.MatchBitap(s, pattern, loc)
}

// MatchBitap locates the best instance of 'pattern' in 'text' near 'loc'
// using the Bitap algorithm.  Returns -1 if no match found.
func (dmp *DMP) MatchBitap(text, pattern string, loc int) int {
	return matchBitap(dmp, text, pattern, loc)
}

//  PATCH FUNCTIONS

// PatchAddContext increases the context until it is unique,
// but doesn't let the pattern expand beyond MatchMaxBits.
func (dmp *DMP) PatchAddContext(p Patch, s string) Patch {
	return patchAddContext(dmp, p, s)
}

func (dmp *DMP) PatchMake(opt ...interface{}) []Patch {
	switch len(opt) {
	case 1:
		diffs, _ := opt[0].([]Diff)
		text1 := DiffText1(diffs)
		return dmp.PatchMake(text1, diffs)

	case 2:
		text1 := opt[0].(string)
		switch t := opt[1].(type) {
		case string:
			diffs := dmp.DiffMain(text1, t, true)
			if len(diffs) > 2 {
				diffs = DiffCleanupSemantic(diffs)
				diffs = dmp.DiffCleanupEfficiency(diffs)
			}
			return dmp.PatchMake(text1, diffs)
		case []Diff:
			return patchMake2(dmp, text1, t)
		}

	case 3:
		return dmp.PatchMake(opt[0], opt[2])
	}
	return []Patch{}
}

// Apply merges a set of patches onto the text.  Returns a patched text,
// as well as an array of true/false values indicating which patches were
// applied.
func (dmp *DMP) Apply(ps []Patch, s string) (string, []bool) {
	if len(ps) == 0 {
		return s, []bool{}
	}

	// Deep copy the patches so that no changes are made to originals.
	ps = PatchDeepCopy(ps)

	nullPadding := patchAddPadding(ps, dmp.PatchMargin)
	s = nullPadding + s + nullPadding
	ps = patchSplitMax(ps, dmp.MatchMaxBits, dmp.PatchMargin)

	x := 0
	// delta keeps track of the offset between the expected and actual
	// location of the previous patch.  If there are patches expected at
	// positions 10 and 20, but the first patch was found at 12, delta is 2
	// and the second patch has an effective expected position of 22.
	delta := 0
	results := make([]bool, len(ps))
	for _, p := range ps {
		expected_loc := p.start2 + delta
		text1 := DiffText1(p.diffs)
		var startLoc int
		endLoc := -1
		if len(text1) > dmp.MatchMaxBits {
			// PatchSplitMax will only provide an oversized pattern
			// in the case of a monster delete.
			startLoc = dmp.MatchMain(
				s, text1[:dmp.MatchMaxBits], expected_loc,
			)
			if startLoc != -1 {
				endLoc = dmp.MatchMain(
					s, text1[len(text1)-dmp.MatchMaxBits:],
					expected_loc+len(text1)-dmp.MatchMaxBits,
				)
				if endLoc == -1 || startLoc >= endLoc {
					// Can't find valid trailing context.  Drop this patch.
					startLoc = -1
				}
			}
		} else {
			startLoc = dmp.MatchMain(s, text1, expected_loc)
		}
		if startLoc == -1 {
			// No match found.  :(
			results[x] = false
			// Subtract the delta for this failed patch from subsequent
			// patches.
			delta -= p.length2 - p.length1
		} else {
			// Found a match.  :)
			results[x] = true
			delta = startLoc - expected_loc
			var text2 string
			if endLoc == -1 {
				text2 = s[startLoc:int(math.Min(float64(startLoc+len(text1)),
					float64(len(s))))]
			} else {
				text2 = s[startLoc:int(math.Min(float64(endLoc+dmp.MatchMaxBits),
					float64(len(s))))]
			}
			if text1 == text2 {
				// Perfect match, just shove the Replacement text in.
				s = s[:startLoc] + DiffText2(p.diffs) +
					s[startLoc+len(text1):]
			} else {
				// Imperfect match.  Run a diff to get a framework of
				// equivalent indices.
				diffs := dmp.DiffMain(text1, text2, false)
				if len(text1) > dmp.MatchMaxBits &&
					float64(DiffLevenshtein(diffs))/float64(len(text1)) >
						dmp.PatchDeleteThreshold {
					// The end points match, but the content is unacceptably
					// bad.
					results[x] = false
				} else {
					diffs = DiffCleanupSemanticLossless(diffs)
					index1 := 0
					for _, d := range p.diffs {
						if d.Type != DiffEqual {
							index2 := DiffXIndex(diffs, index1)
							if d.Type == DiffInsert {
								// Insertion
								s = s[:startLoc+index2] +
									d.Text + s[startLoc+index2:]
							} else if d.Type == DiffDelete {
								// Deletion
								startIndex := startLoc + index2
								s = s[:startIndex] +
									s[startIndex+DiffXIndex(
										diffs,
										index1+len(d.Text),
									)-index2:]
							}
						}
						if d.Type != DiffDelete {
							index1 += len(d.Text)
						}
					}
				}
			}
		}
		x++
	}
	// Strip the padding off.
	s = s[len(nullPadding) : len(nullPadding)+(len(s)-2*len(nullPadding))]
	return s, results
}

// PatchAddPadding adds some padding on text start and end so that edges can
// match something.  Intended to be called only from within patch_apply.
func (dmp *DMP) PatchAddPadding(ps []Patch) string {
	return patchAddPadding(ps, dmp.PatchMargin)
}

// PatchSplitMax looks through the patches and breaks up any which are longer
// than the maximum limit of the match algorithm.
// Intended to be called only from within patch_apply.
func (dmp *DMP) PatchSplitMax(ps []Patch) []Patch {
	return patchSplitMax(ps, dmp.MatchMaxBits, dmp.PatchMargin)
}

package dmp

import (
	"unicode/utf8"
)

// DiffCleanupSemanticLossless looks for single edits surrounded on both
// sides by equalities which can be shifted sideways to align the edit to a
// word boundary.
// e.g: The c<ins>at c</ins>ame. -> The <ins>cat </ins>came.
func DiffCleanupSemanticLossless(diffs []Diff) []Diff {
	/**
	 * Given two strings, compute a score representing whether the internal
	 * boundary falls on logical boundaries.
	 * Scores range from 6 (best) to 0 (worst).
	 * Closure, but does not reference any external variables.
	 * @param {string} one First string.
	 * @param {string} two Second string.
	 * @return {number} The score.
	 * @private
	 */
	diffCleanupSemanticScore := func(one, two string) int {
		if len(one) == 0 || len(two) == 0 {
			// Edges are the best.
			return 6
		}

		// Each port of this function behaves slightly differently due to
		// subtle differences in each language's definition of things like
		// 'whitespace'.  Since this function's purpose is largely cosmetic,
		// the choice has been made to use each language's native features
		// rather than force total conformity.
		rune1, _ := utf8.DecodeLastRuneInString(one)
		rune2, _ := utf8.DecodeRuneInString(two)
		char1 := string(rune1)
		char2 := string(rune2)

		nonAlphaNumeric1 := nonAlphaNumericRegex_.MatchString(char1)
		nonAlphaNumeric2 := nonAlphaNumericRegex_.MatchString(char2)
		whitespace1 := nonAlphaNumeric1 && whitespaceRegex_.MatchString(char1)
		whitespace2 := nonAlphaNumeric2 && whitespaceRegex_.MatchString(char2)
		lineBreak1 := whitespace1 && linebreakRegex_.MatchString(char1)
		lineBreak2 := whitespace2 && linebreakRegex_.MatchString(char2)
		blankLine1 := lineBreak1 && blanklineEndRegex_.MatchString(one)
		blankLine2 := lineBreak2 && blanklineEndRegex_.MatchString(two)

		if blankLine1 || blankLine2 {
			// Five points for blank lines.
			return 5
		} else if lineBreak1 || lineBreak2 {
			// Four points for line breaks.
			return 4
		} else if nonAlphaNumeric1 && !whitespace1 && whitespace2 {
			// Three points for end of sentences.
			return 3
		} else if whitespace1 || whitespace2 {
			// Two points for whitespace.
			return 2
		} else if nonAlphaNumeric1 || nonAlphaNumeric2 {
			// One point for non-alphanumeric.
			return 1
		}
		return 0
	}

	pointer := 1

	// Intentionally ignore the first and last element (don't need checking).
	for pointer < len(diffs)-1 {
		if diffs[pointer-1].Type == DiffEqual &&
			diffs[pointer+1].Type == DiffEqual {

			// This is a single edit surrounded by equalities.
			equality1 := diffs[pointer-1].Text
			edit := diffs[pointer].Text
			equality2 := diffs[pointer+1].Text

			// First, shift the edit as far left as possible.
			commonOffset := DiffCommonSuffix(equality1, edit)
			if commonOffset > 0 {
				commonString := edit[len(edit)-commonOffset:]
				equality1 = equality1[0 : len(equality1)-commonOffset]
				edit = commonString + edit[:len(edit)-commonOffset]
				equality2 = commonString + equality2
			}

			// Second, step character by character right, looking for the best
			// fit.
			bestEquality1 := equality1
			bestEdit := edit
			bestEquality2 := equality2
			bestScore := diffCleanupSemanticScore(equality1, edit) +
				diffCleanupSemanticScore(edit, equality2)

			for len(edit) != 0 && len(equality2) != 0 {
				_, sz := utf8.DecodeRuneInString(edit)
				if len(equality2) < sz || edit[:sz] != equality2[:sz] {
					break
				}
				equality1 += edit[:sz]
				edit = edit[sz:] + equality2[:sz]
				equality2 = equality2[sz:]
				score := diffCleanupSemanticScore(equality1, edit) +
					diffCleanupSemanticScore(edit, equality2)
					// The >= encourages trailing rather than leading
					// whitespace on edits.
				if score >= bestScore {
					bestScore = score
					bestEquality1 = equality1
					bestEdit = edit
					bestEquality2 = equality2
				}
			}

			if diffs[pointer-1].Text != bestEquality1 {
				// We have an improvement, save it back to the diff.
				if len(bestEquality1) != 0 {
					diffs[pointer-1].Text = bestEquality1
				} else {
					diffs = splice(diffs, pointer-1, 1)
					pointer--
				}

				diffs[pointer].Text = bestEdit
				if len(bestEquality2) != 0 {
					diffs[pointer+1].Text = bestEquality2
				} else {
					diffs = append(diffs[:pointer+1], diffs[pointer+2:]...)
					pointer--
				}
			}
		}
		pointer++
	}

	return diffs
}

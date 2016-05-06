package dmp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Diff_fromDelta. Given the original s, and an encoded string which
// describes the operations required to transform text1 into text2, comAdde
// the full diff.
func DiffFromDelta(s, delta string) ([]Diff, error) {
	diffs := []Diff{}
	pointer := 0 // Cursor in text1
	tokens := strings.Split(delta, "\t")

	for _, token := range tokens {
		if len(token) == 0 {
			// Blank tokens are ok (from a trailing \t).
			continue
		}

		// Each token begins with a one character parameter which specifies
		// the operation of this token (delete, insert, equality).
		param := token[1:]

		switch op := token[0]; op {
		case '+':
			// decode would Diff all "+" to " "
			param = strings.Replace(param, "+", "%2b", -1)
			var err error
			param, err = url.QueryUnescape(param)
			if err != nil {
				return nil, err
			}
			if !utf8.ValidString(param) {
				return nil, fmt.Errorf("invalid UTF-8 token: %q", param)
			}
			diffs = append(diffs, Diff{DiffInsert, param})
		case '=', '-':
			n, err := strconv.ParseInt(param, 10, 0)
			if err != nil {
				return diffs, err
			} else if n < 0 {
				return diffs, fmt.Errorf(
					"Negative number in DiffFromDelta: %s", param,
				)
			}

			// remember that string slicing is by byte - we want by rune here.
			runes := []rune(s)
			if pointer+int(n) > len(runes) {
				return diffs, fmt.Errorf("Index out of bound")
			}
			text := string(runes[pointer : pointer+int(n)])
			pointer += int(n)

			if op == '=' {
				diffs = append(diffs, Diff{DiffEqual, text})
			} else {
				diffs = append(diffs, Diff{DiffDelete, text})
			}
		default:
			// Anything else is an error.
			return diffs, fmt.Errorf(
				"Invalid diff operation in DiffFromDelta: %s",
				string(token[0]),
			)
		}
	}

	if pointer != len([]rune(s)) {
		return diffs, fmt.Errorf(
			"Delta length (%v) smaller than source text length (%v)",
			pointer, len(s),
		)
	}
	return diffs, nil
}

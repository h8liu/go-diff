package diffmatchpatch

// commonPrefixLength returns the length of the common prefix of two rune
// slices.
func commonPrefixLength(text1, text2 []rune) int {
	short, long := text1, text2
	if len(short) > len(long) {
		short, long = long, short
	}
	for i, r := range short {
		if r != long[i] {
			return i
		}
	}
	return len(short)
}

// commonSuffixLength returns the length of the common suffix of two rune
// slices.
func commonSuffixLength(text1, text2 []rune) int {
	n := min(len(text1), len(text2))
	for i := 0; i < n; i++ {
		if text1[len(text1)-i-1] != text2[len(text2)-i-1] {
			return i
		}
	}
	return n
}

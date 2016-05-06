package dmp

// Diff represents one diff operation
type Diff struct {
	Type Operation
	Text string
}

func diffEq(s string) Diff { return Diff{DiffEqual, s} }

func diffPrepend(head Diff, diffs []Diff) []Diff {
	ret := make([]Diff, 0, len(diffs)+1)
	ret = append(ret, head)
	ret = append(ret, diffs...)
	return ret
}

func diffAppend(diffs []Diff, tail Diff) []Diff {
	return append(diffs, tail)
}

package diffmatchpatch

// Diff represents one diff operation
type Diff struct {
	Type Operation
	Text string
}

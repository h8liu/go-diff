package dmp

import (
	"regexp"
)

// Define some regex patterns for matching boundaries.
var (
	nonAlphaNumericRegex_ = regexp.MustCompile(`[^a-zA-Z0-9]`)
	whitespaceRegex_      = regexp.MustCompile(`\s`)
	linebreakRegex_       = regexp.MustCompile(`[\r\n]`)
	blanklineEndRegex_    = regexp.MustCompile(`\n\r?\n$`)
	blanklineStartRegex_  = regexp.MustCompile(`^\r?\n\r?\n`)
)

package dmp

import (
	"bytes"
	"html"
	"strings"
)

// DiffPrettyHtml converts a []Diff into a pretty HTML report.
// It is intended as an example from which to write one's own
// display functions.
func DiffPrettyHtml(diffs []Diff) string {
	var buf bytes.Buffer
	for _, d := range diffs {
		text := strings.Replace(
			html.EscapeString(d.Text), "\n", "&para;<br>", -1,
		)
		switch d.Type {
		case DiffInsert:
			buf.WriteString("<ins style=\"background:#e6ffe6;\">")
			buf.WriteString(text)
			buf.WriteString("</ins>")
		case DiffDelete:
			buf.WriteString("<del style=\"background:#ffe6e6;\">")
			buf.WriteString(text)
			buf.WriteString("</del>")
		case DiffEqual:
			buf.WriteString("<span>")
			buf.WriteString(text)
			buf.WriteString("</span>")
		}
	}
	return buf.String()
}

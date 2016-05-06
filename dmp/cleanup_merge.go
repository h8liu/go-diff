package dmp

import (
	"strings"
)

// DiffCleanupMerge reorders and merges like edit sections.  Merge
// equalities.  Any edit section can move as long as it doesn't cross an
// equality.
func DiffCleanupMerge(ds []Diff) []Diff {
	// Add a dummy entry at the end.
	ds = append(ds, Diff{DiffEqual, ""})
	i := 0
	ndel := 0
	nins := 0
	commonlength := 0
	delStr := ""
	insStr := ""

	for i < len(ds) {
		switch ds[i].Type {
		case DiffInsert:
			nins += 1
			insStr += ds[i].Text
			i += 1
			break
		case DiffDelete:
			ndel += 1
			delStr += ds[i].Text
			i += 1
			break
		case DiffEqual:
			// Upon reaching an equality, check for prior redundancies.
			if ndel+nins > 1 {
				if ndel != 0 && nins != 0 {
					// Factor out any common prefixies.
					commonlength = DiffCommonPrefix(
						insStr, delStr,
					)
					if commonlength != 0 {
						x := i - ndel - nins
						if x > 0 && ds[x-1].Type == DiffEqual {
							ds[x-1].Text += insStr[:commonlength]
						} else {
							ds = append(
								[]Diff{
									{DiffEqual,
										insStr[:commonlength]},
								},
								ds...,
							)
							i += 1
						}
						insStr = insStr[commonlength:]
						delStr = delStr[commonlength:]
					}
					// Factor out any common suffixies.
					commonlength = DiffCommonSuffix(
						insStr, delStr,
					)
					if commonlength != 0 {
						insert_index := len(insStr) - commonlength
						delete_index := len(delStr) - commonlength
						ds[i].Text =
							insStr[insert_index:] + ds[i].Text
						insStr = insStr[:insert_index]
						delStr = delStr[:delete_index]
					}
				}
				// Delete the offending records and add the merged ones.
				if ndel == 0 {
					ds = splice(ds, i-nins,
						ndel+nins,
						Diff{DiffInsert, insStr})
				} else if nins == 0 {
					ds = splice(ds, i-ndel,
						ndel+nins,
						Diff{DiffDelete, delStr})
				} else {
					ds = splice(
						ds, i-ndel-nins,
						ndel+nins,
						Diff{DiffDelete, delStr},
						Diff{DiffInsert, insStr},
					)
				}

				i = i - ndel - nins + 1
				if ndel != 0 {
					i += 1
				}
				if nins != 0 {
					i += 1
				}
			} else if i != 0 && ds[i-1].Type == DiffEqual {
				// Merge this equality with the previous one.
				ds[i-1].Text += ds[i].Text
				ds = append(ds[:i], ds[i+1:]...)
			} else {
				i++
			}
			nins = 0
			ndel = 0
			delStr = ""
			insStr = ""
			break
		}
	}

	if len(ds[len(ds)-1].Text) == 0 {
		ds = ds[0 : len(ds)-1] // Remove the dummy entry at the end.
	}

	// Second pass: look for single edits surrounded on both sides by
	// equalities which can be shifted sideways to eliminate an equality.
	// e.g: A<ins>BA</ins>C -> <ins>AB</ins>AC
	changes := false
	i = 1
	// Intentionally ignore the first and last element (don't need checking).
	for i < (len(ds) - 1) {
		if ds[i-1].Type == DiffEqual &&
			ds[i+1].Type == DiffEqual {
			// This is a single edit surrounded by equalities.
			if strings.HasSuffix(ds[i].Text, ds[i-1].Text) {
				// Shift the edit over the previous equality.
				ds[i].Text = ds[i-1].Text +
					ds[i].Text[:len(ds[i].Text)-len(ds[i-1].Text)]
				ds[i+1].Text =
					ds[i-1].Text + ds[i+1].Text
				ds = splice(ds, i-1, 1)
				changes = true
			} else if strings.HasPrefix(
				ds[i].Text, ds[i+1].Text,
			) {
				// Shift the edit over the next equality.
				ds[i-1].Text += ds[i+1].Text
				ds[i].Text =
					ds[i].Text[len(ds[i+1].Text):] +
						ds[i+1].Text
				ds = splice(ds, i+1, 1)
				changes = true
			}
		}
		i++
	}

	// If shifts were made, the diff needs reordering and another shift sweep.
	if changes {
		ds = DiffCleanupMerge(ds)
	}

	return ds
}

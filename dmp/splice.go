package dmp

func splice(slice []Diff, index int, amount int, elements ...Diff) []Diff {
	return append(slice[:index], append(elements, slice[index+amount:]...)...)
}

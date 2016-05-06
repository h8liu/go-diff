package dmp

type Operation int8

const (
	DiffDelete Operation = -1
	DiffInsert Operation = 1
	DiffEqual  Operation = 0
)

package main

const (
	_ = iota

	PrecGroup
	PrecAssign

	PrecPrefix

	PrecSum
	PrecMul

	PrecOr
	PrecXor
	PrecAnd
	PrecShift

	PrecCall
	PrecLiteral
)

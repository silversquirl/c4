package main

const (
	_ = iota

	PrecGroup
	PrecAssign

	PrecOr
	PrecXor
	PrecAnd
	PrecShift

	PrecMul
	PrecSum

	PrecPrefix

	PrecCall
	PrecLiteral
)

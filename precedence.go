package main

const (
	_ = iota

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

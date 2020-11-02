package main

const (
	_ = iota

	PrecOr
	PrecXor
	PrecAnd
	PrecShift

	PrecMul
	PrecSum

	PrecCall
	PrecLiteral
)

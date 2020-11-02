//go:generate stringer -type PrefixOperator -linecomment
//go:generate stringer -type BinaryOperator -linecomment
package main

const (
	_ PrefixOperator = iota

	PrefNot // !
	PrefInv // ^
	PrefNeg // -
	PrefPos // +

	PrefixOperatorMax
)

const (
	_ BinaryOperator = iota

	BinAdd // +
	BinSub // -
	BinMul // *
	BinDiv // /
	BinMod // %

	BinOr  // |
	BinXor // ^
	BinAnd // &
	BinShl // <<
	BinShr // >>

	BinaryOperatorMax
)

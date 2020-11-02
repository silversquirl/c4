// Code generated by "stringer -type TokenType -linecomment"; DO NOT EDIT.

package main

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[TEOF-0]
	_ = x[TComment-1]
	_ = x[TSpace-2]
	_ = x[TNewline-3]
	_ = x[TSemi-4]
	_ = x[TComma-5]
	_ = x[TLParen-6]
	_ = x[TRParen-7]
	_ = x[TLSquare-8]
	_ = x[TRSquare-9]
	_ = x[TLBrace-10]
	_ = x[TRBrace-11]
	_ = x[TEquals-12]
	_ = x[TPlus-13]
	_ = x[TMinus-14]
	_ = x[TAster-15]
	_ = x[TSlash-16]
	_ = x[TPerc-17]
	_ = x[TExcl-18]
	_ = x[TPipe-19]
	_ = x[TCaret-20]
	_ = x[TAmp-21]
	_ = x[TLess-22]
	_ = x[TGreater-23]
	_ = x[TShl-24]
	_ = x[TShr-25]
	_ = x[TLand-26]
	_ = x[TLor-27]
	_ = x[TCeq-28]
	_ = x[TCne-29]
	_ = x[TCle-30]
	_ = x[TCge-31]
	_ = x[TKelse-32]
	_ = x[TKextern-33]
	_ = x[TKfn-34]
	_ = x[TKfor-35]
	_ = x[TKif-36]
	_ = x[TKpub-37]
	_ = x[TKreturn-38]
	_ = x[TKtype-39]
	_ = x[TKvar-40]
	_ = x[TIdent-41]
	_ = x[TType-42]
	_ = x[TString-43]
	_ = x[TInteger-44]
	_ = x[TFloat-45]
	_ = x[TInvalid-46]
	_ = x[TokenTypeMax-47]
}

const _TokenType_name = "end of filecommentwhitespacenewline';'',''('')''['']''{''}''=''+''-''*''/''%''!''|''^''&''<''>''<<''>>''&&''||''==''!=''<=''>=''else''extern''fn''for''if''pub''return''type''var'identifiertype namestring literalinteger literalfloat literalinvalid tokenTokenTypeMax"

var _TokenType_index = [...]uint16{0, 11, 18, 28, 35, 38, 41, 44, 47, 50, 53, 56, 59, 62, 65, 68, 71, 74, 77, 80, 83, 86, 89, 92, 95, 99, 103, 107, 111, 115, 119, 123, 127, 133, 141, 145, 150, 154, 159, 167, 173, 178, 188, 197, 211, 226, 239, 252, 264}

func (i TokenType) String() string {
	if i < 0 || i >= TokenType(len(_TokenType_index)-1) {
		return "TokenType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TokenType_name[_TokenType_index[i]:_TokenType_index[i+1]]
}

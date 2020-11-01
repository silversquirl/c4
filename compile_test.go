package main

import (
	"strings"
	"testing"
)

func spc(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func testCompile(t *testing.T, prog Program, ir string) {
	b := &strings.Builder{}
	c := NewCompiler(b)
	c.Compile(prog)

	// Compare without taking into account indentation
	ir0 := []byte(b.String())
	ir1 := []byte(ir)
	var i0, i1 int
	for {
		spc0 := false
		for i0 < len(ir0) && spc(ir0[i0]) {
			i0++
			spc0 = true
		}
		spc1 := false
		for i1 < len(ir1) && spc(ir1[i1]) {
			i1++
			spc1 = true
		}

		end0 := i0 >= len(ir0)
		end1 := i1 >= len(ir1)
		if end0 && end1 {
			return
		} else if end0 || end1 {
			t.Fatal("Generated and expected IRs are of different length")
		}

		if i0 == 0 {
			spc0 = true
		}
		if i1 == 0 {
			spc1 = true
		}

		if spc0 != spc1 || ir0[i0] != ir1[i1] {
			t.Fatalf("Generated and expected IRs do not match at bytes %d, %d\n%s", i0, i1, b)
		}
		i0++
		i1++
	}
}

func testMainCompile(t *testing.T, stmts []Statement, ir string) {
	ir = `export function w $main() { @start ` + ir + ` }`
	testCompile(t, Program{Function{true, "main", TypeI32, nil, stmts}}, ir)
}

func TestReturn0(t *testing.T) {
	/*
		fn main() I32 {
			return 0
		}
	*/
	testMainCompile(t, []Statement{ReturnStmt{IntegerExpr("0")}}, `ret 0`)
}

// TODO: test unsigned div, mod and shr
func TestArithmetic(t *testing.T) {
	/*
		fn main() I32 {
			4 + 2
			4 - 2
			4 * 2
			4 / 2
			4 % 2

			4 | 2
			4 ^ 2
			4 & 2
			4 << 2
			4 >> 2

			return 0
		}
	*/
	bin := func(op BinaryOperator) Statement {
		return ExprStmt{BinaryExpr{op, IntegerExpr("4"), IntegerExpr("2")}}
	}
	testMainCompile(t, []Statement{
		bin(BOpAdd), bin(BOpSub),
		bin(BOpMul), bin(BOpDiv), bin(BOpMod),

		bin(BOpOr), bin(BOpXor), bin(BOpAnd),
		bin(BOpShl), bin(BOpShr),

		ReturnStmt{IntegerExpr("0")},
	}, `
		%t1 =l add 4, 2
		%t2 =l sub 4, 2
		%t3 =l mul 4, 2
		%t4 =l div 4, 2
		%t5 =l rem 4, 2

		%t6  =l or  4, 2
		%t7  =l xor 4, 2
		%t8  =l and 4, 2
		%t9  =l shl 4, 2
		%t10 =l sar 4, 2

		ret 0
	`)
}

func TestNestedArithmetic(t *testing.T) {
	/*
		fn main() I32 {
			return (1 + 10*2) * 2
		}
	*/
	testMainCompile(t, []Statement{
		ReturnStmt{
			BinaryExpr{BOpMul,
				BinaryExpr{BOpAdd,
					IntegerExpr("1"),
					BinaryExpr{BOpMul,
						IntegerExpr("10"),
						IntegerExpr("2"),
					},
				},
				IntegerExpr("2"),
			},
		},
	}, `
		%t1 =l mul 10, 2
		%t2 =l add 1, %t1
		%t3 =l mul %t2, 2
		ret %t3
	`)
}

func TestVariables(t *testing.T) {
	/*
		extern global I32
		fn main() I32 {
			var i I32
			var j I32
			i = 7
			j = 5
			i = i + j
			return i + global
		}
	*/
	testCompile(t, Program{
		VarDecl{"global", TypeI32},
		Function{true, "main", TypeI32, nil, []Statement{
			VarDecl{"i", TypeI32},
			VarDecl{"j", TypeI32},
			ExprStmt{AssignExpr{VarExpr("i"), IntegerExpr("7")}},
			ExprStmt{AssignExpr{VarExpr("j"), IntegerExpr("5")}},
			ExprStmt{AssignExpr{VarExpr("i"), BinaryExpr{BOpAdd, VarExpr("i"), VarExpr("j")}}},
			ReturnStmt{BinaryExpr{BOpAdd, VarExpr("i"), VarExpr("global")}},
		}},
	}, `
		export function w $main() {
		@start
			%t1 =l alloc4 4
			%t2 =l alloc4 4
			storew 7, %t1
			storew 5, %t2

			%t3 =w loadw %t1
			%t4 =w loadw %t2
			%t5 =w add %t3, %t4
			storew %t5, %t1

			%t6 =w loadw %t1
			%t7 =w loadw $global
			%t8 =w add %t6, %t7
			ret %t8
		}
	`)
}

func TestReferenceVariable(t *testing.T) {
	/*
		fn main() I32 {
			var i I32
			var j [I32]
			j = &i
			return 0
		}
	*/
	testMainCompile(t, []Statement{
		VarDecl{"i", TypeI32},
		VarDecl{"j", PointerTo(TypeI32)},
		ExprStmt{AssignExpr{VarExpr("j"), RefExpr{VarExpr("i")}}},
		ReturnStmt{IntegerExpr("0")},
	}, `
		%t1 =l alloc4 4
		%t2 =l alloc8 8
		storel %t1, %t2
		ret 0
	`)
}

func TestFunctionCall(t *testing.T) {
	/*
		extern printi fn(I64)
		fn main() I32 {
			printi(42)
			return 0
		}
	*/
	testCompile(t, Program{
		VarDecl{"printi", FuncType{[]ConcreteType{TypeI64}, nil}},
		Function{true, "main", TypeI32, nil, []Statement{
			ExprStmt{CallExpr{VarExpr("printi"), []Expression{IntegerExpr("42")}}},
			ReturnStmt{IntegerExpr("0")},
		}},
	}, `
		export function w $main() {
		@start
			call $printi(l 42)
			ret 0
		}
	`)
}

func TestStringLiteral(t *testing.T) {
	/*
		extern puts fn([I8]) I32
		fn main() I32 {
			puts("str0")
			puts("str0")
			puts("str1")
			puts("str1")
			puts("str2")
			puts("str2")
			return 0
		}
	*/
	puts := func(s string) Statement {
		return ExprStmt{CallExpr{VarExpr("puts"), []Expression{StringExpr(s)}}}
	}
	testCompile(t, Program{
		VarDecl{"puts", FuncType{[]ConcreteType{PointerTo(TypeI8)}, TypeI32}},
		Function{true, "main", TypeI32, nil, []Statement{
			puts("str0"),
			puts("str0"),
			puts("str1"),
			puts("str1"),
			puts("str2"),
			puts("str2"),
			ReturnStmt{IntegerExpr("0")},
		}},
	}, `
		export function w $main() {
		@start
			%t1 =w call $puts(l $str0)
			%t2 =w call $puts(l $str0)
			%t3 =w call $puts(l $str1)
			%t4 =w call $puts(l $str1)
			%t5 =w call $puts(l $str2)
			%t6 =w call $puts(l $str2)
			ret 0
		}
		data $str0 = { b "str0", b 0 }
		data $str1 = { b "str1", b 0 }
		data $str2 = { b "str2", b 0 }
	`)
}

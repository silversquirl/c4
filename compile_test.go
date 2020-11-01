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

func TestReturn0(t *testing.T) {
	/*
		fn main() I32 {
			return 0
		}
	*/
	testCompile(t, Program{
		Function{true, "main", TypeI32, nil, []Statement{
			ReturnStmt{IntegerExpr("0")},
		}},
	}, `
		export function w $main() {
		@start
			ret 0
		}
	`)
}

func TestReturnArith(t *testing.T) {
	/*
		fn main() I32 {
			return (1 + 10*2) * 2
		}
	*/
	testCompile(t, Program{
		Function{true, "main", TypeI32, nil, []Statement{
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
		}},
	}, `
		export function w $main() {
		@start
			%t1 =l mul 10, 2
			%t2 =l add 1, %t1
			%t3 =l mul %t2, 2
			ret %t3
		}
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
			puts("Hello, world!")
			return 0
		}
	*/
	testCompile(t, Program{
		VarDecl{"puts", FuncType{[]ConcreteType{PointerTo(TypeI8)}, TypeI32}},
		Function{true, "main", TypeI32, nil, []Statement{
			ExprStmt{CallExpr{VarExpr("puts"), []Expression{StringExpr("Hello, world!")}}},
			ReturnStmt{IntegerExpr("0")},
		}},
	}, `
		export function w $main() {
		@start
			%t1 =w call $puts(l $str0)
			ret 0
		}
		data $str0 = { b "Hello, world!", b 0 }
	`)
}

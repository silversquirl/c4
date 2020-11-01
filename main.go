package main

import "os"

func main() {
	prog := Program{
		VarDecl{"puts", FuncType{[]ConcreteType{PointerTo(TypeI8)}, TypeI32}},
		// These type signatures are lies, but we don't support variadics yet
		VarDecl{"scanf", FuncType{[]ConcreteType{PointerTo(TypeI8), PointerTo(TypeI32), TypeI32}, TypeI32}},
		VarDecl{"printf", FuncType{[]ConcreteType{PointerTo(TypeI8), TypeI32, TypeI32, TypeI32}, TypeI32}},

		Function{true, "main", TypeI32, nil, []Statement{
			// puts("Enter two numbers:")
			//
			// var a, b I32
			// scanf("%d", &a)
			// scanf("%d", &b)
			//
			// printf("%d + %d = %d\n", a, b, a+b)
			// return 0
			ExprStmt{CallExpr{VarExpr("puts"), []Expression{StringExpr("Enter two numbers:")}}},

			VarDecl{"a", TypeI32},
			VarDecl{"b", TypeI32},
			ExprStmt{CallExpr{VarExpr("scanf"), []Expression{StringExpr("%d"), RefExpr{VarExpr("a")}}}},
			ExprStmt{CallExpr{VarExpr("scanf"), []Expression{StringExpr("%d"), RefExpr{VarExpr("b")}}}},

			ExprStmt{CallExpr{VarExpr("printf"), []Expression{
				StringExpr("%d + %d = %d\n"),
				VarExpr("a"), VarExpr("b"),
				BinaryExpr{BOpAdd, VarExpr("a"), VarExpr("b")},
			}}},
			ReturnStmt{IntegerExpr("0")},
		}},
	}
	c := NewCompiler(os.Stdout)
	c.Compile(prog)
}

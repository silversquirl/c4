package main

import "os"

func main() {
	prog := Program{
		VarDecl{"puts", FuncType{[]ConcreteType{PointerTo(TypeI8)}, TypeI32}},
		// VarDecl{"global", TypeI64},
		Function{true, "main", TypeI32, nil, []Statement{
			/*
				// return 0
				ReturnStmt{IntegerExpr("0")},
			*/

			/*
				// return (1 + 10*2) * 2
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
			*/

			/*
				// var i I64
				// var j I64
				// i = 7
				// j = 5
				// i = i + j
				// return i + global
				VarDecl{"i", TypeI64},
				VarDecl{"j", TypeI64},
				ExprStmt{AssignExpr{VarExpr("i"), IntegerExpr("7")}},
				ExprStmt{AssignExpr{VarExpr("j"), IntegerExpr("5")}},
				ExprStmt{AssignExpr{VarExpr("i"), BinaryExpr{BOpAdd, VarExpr("i"), VarExpr("j")}}},
				ReturnStmt{BinaryExpr{BOpAdd, VarExpr("i"), VarExpr("global")}},
			*/

			// /*
			// printi(42)
			// return 0
			ExprStmt{CallExpr{VarExpr("puts"), []Expression{StringExpr("Hello, world!")}}},
			ReturnStmt{IntegerExpr("0")},
			// */
		}},
	}
	c := NewCompiler(os.Stdout)
	c.Compile(prog)
}

package main

import "os"

func main() {
	prog := Program{
		{true, "main", TypeI32, nil, []Statement{
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

			// /*
			// var i I64
			// var j I64
			// i = 7
			// j = 5
			// return i + j
			DeclStmt{"i", TypeI64},
			DeclStmt{"j", TypeI64},
			ExprStmt{AssignExpr{VarExpr("i"), IntegerExpr("7")}},
			ExprStmt{AssignExpr{VarExpr("j"), IntegerExpr("5")}},
			ReturnStmt{BinaryExpr{BOpAdd, VarExpr("i"), VarExpr("j")}},
			// */
		}},
	}
	c := NewCompiler(os.Stdout)
	prog.GenIR(c)
}

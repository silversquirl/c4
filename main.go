package main

import "os"

func main() {
	prog := Program{
		{true, "main", TypeI32, nil, []Statement{
			//ReturnStmt{IntegerExpr("0")},
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
		}},
	}
	c := NewCompiler(os.Stdout)
	prog.GenIR(c)
}

package main

import "os"

func main() {
	prog := Program{
		VarDecl{"puts", FuncType{[]ConcreteType{PointerTo(TypeI8)}, TypeI32}},
		Function{true, "main", TypeI32, nil, []Statement{
			// puts("Hello, world!")
			// return 0
			ExprStmt{CallExpr{VarExpr("puts"), []Expression{StringExpr("Hello, world!")}}},
			ReturnStmt{IntegerExpr("0")},
		}},
	}
	c := NewCompiler(os.Stdout)
	c.Compile(prog)
}

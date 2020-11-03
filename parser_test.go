package main

import (
	"reflect"
	"testing"
)

func testParser(code string) *parser {
	toks := make(chan Token)
	go Tokenize(code, toks)
	return &parser{<-toks, toks}
}

func testParse(t *testing.T, code string, prog Program) {
	prog2, err := Parse(code)
	t.Log(prog)
	t.Log(prog2)
	if err != nil {
		t.Fatal("Parse error:", err)
	}
	checkParse(t, prog, prog2)
}

func checkParse(t *testing.T, a, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Fatal("Parse trees differ")
	}
}

func TestExpression(t *testing.T) {
	expr := IntegerExpr("0")
	expr2 := testParser("0").parseExpression(0)
	checkParse(t, expr, expr2)
}

func TestStatement(t *testing.T) {
	expr := ReturnStmt{IntegerExpr("0")}
	expr2 := testParser("return 0").parseStatement()
	checkParse(t, expr, expr2)
}

func TestMinimal(t *testing.T) {
	testParse(t, `
		pub fn main() I32 {
			return 0
		}
	`, Program{
		Function{true, "main", nil, NamedTypeExpr("I32"), []Statement{
			ReturnStmt{IntegerExpr("0")},
		}},
	})
}

func TestHelloWorld(t *testing.T) {
	testParse(t, `
		pub fn main() I32 {
			puts("Hello, world!")
			return 0
		}
	`, Program{
		Function{true, "main", nil, NamedTypeExpr("I32"), []Statement{
			ExprStmt{CallExpr{VarExpr("puts"), []Expression{StringExpr("Hello, world!")}}},
			ReturnStmt{IntegerExpr("0")},
		}},
	})
}

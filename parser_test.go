package main

import (
	"testing"
)

func testParser(code string) *parser {
	toks := make(chan Token)
	go Tokenize(code, toks)
	return &parser{<-toks, toks}
}

func checkParse(t *testing.T, a, b string) {
	if eq, ai, bi := CodeCompare(a, b); !eq {
		t.Errorf("Generated and expected code does not match at bytes %d, %d\n%s!!%s", ai, bi, a[:ai], a[ai:])
	}
}

func testProg(t *testing.T, code, expect string) {
	prog := testParser(code).parseProgram()
	checkParse(t, prog.Format(0), expect)
}

func testStmt(t *testing.T, code, expect string) {
	stmt := testParser(code).parseStatement()
	checkParse(t, stmt.Format(0), expect)
}

func testExpr(t *testing.T, code, expect string) {
	expr := testParser(code).parseExpression(0)
	checkParse(t, expr.Format(0), expect)
}

func TestStatement(t *testing.T) {
	testStmt(t, "return 0", "return 0")
}

func TestExpression(t *testing.T) {
	testExpr(t, "0", "0")
}

func TestMinimal(t *testing.T) {
	testProg(t, `
		pub fn main() I32 {
			return 0
		}
	`, `
		pub fn main() I32 {
			return 0
		}
	`)
}

func TestHelloWorld(t *testing.T) {
	testProg(t, `
		pub fn main() I32 {
			puts("Hello, world!")
			return 0
		}
	`, `
		pub fn main() I32 {
			puts("Hello, world!")
			return 0
		}
	`)
}

func TestInfixPrefix(t *testing.T) {
	testExpr(t, "1+1", "(1 + 1)")
	testExpr(t, "1-1", "(1 - 1)")
}

func TestPrecedence(t *testing.T) {
	testExpr(t, "-f(a, b)", "-(f(a, b))")
	testExpr(t, "-a * b", "(-(a) * b)")
	testExpr(t, "a + f(a, b)", "(a + f(a, b))")
	testExpr(t, "a + b * c", "(a + (b * c))")
	testExpr(t, "a << b + c", "(a << (b + c))")
	testExpr(t, "a & b << c", "(a & (b << c))")
	testExpr(t, "a == b & c", "(a == (b & c))")
	testExpr(t, "a && b == c", "(a && (b == c))")
	testExpr(t, "a || b && c", "(a || (b && c))")
	testExpr(t, "a = b || c", "(a = (b || c))")
	testExpr(t, "(a = b) || c", "((a = b) || c)")
}

func TestAssociativity(t *testing.T) {
	testExpr(t, "a + b + c", "((a + b) + c)")
	testExpr(t, "a + b - c", "((a + b) - c)")
	testExpr(t, "a * b * c", "((a * b) * c)")
	testExpr(t, "a * b / c", "((a * b) / c)")
	testExpr(t, "a << b << c", "((a << b) << c)")
	testExpr(t, "a << b >> c", "((a << b) >> c)")
	testExpr(t, "a & b & c", "((a & b) & c)")
	testExpr(t, "a ^ b & c", "((a ^ b) & c)")
	testExpr(t, "a | b ^ c", "((a | b) ^ c)")
	testExpr(t, "a == b == c", "((a == b) == c)")
	testExpr(t, "a == b != c", "((a == b) != c)")
	testExpr(t, "a == b < c", "((a == b) < c)")
	testExpr(t, "a && b && c", "((a && b) && c)")
	testExpr(t, "a || b || c", "((a || b) || c)")
	testExpr(t, "a = b = c", "(a = (b = c))")
}

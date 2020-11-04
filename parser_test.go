package main

import (
	"testing"
)

func testParser(code string) *parser {
	toks := make(chan Token)
	go Tokenize(code, toks)
	return &parser{<-toks, toks}
}

func testParse(t *testing.T, code, expect string) {
	prog, err := Parse(code)
	if err != nil {
		t.Fatal("Parse error:", err)
	}
	checkParse(t, prog.Format(0), expect)
}

func checkParse(t *testing.T, a, b string) {
	if eq, ai, bi := CodeCompare(a, b); !eq {
		t.Fatalf("Generated and expected code does not match at bytes %d, %d\n%s!!%s", ai, bi, a[:ai], a[ai:])
	}
}

func TestExpression(t *testing.T) {
	expr := testParser("0").parseExpression(0)
	checkParse(t, "0", expr.Format(0))
}

func TestStatement(t *testing.T) {
	stmt := testParser("return 0").parseStatement()
	checkParse(t, "return 0", stmt.Format(0))
}

func TestMinimal(t *testing.T) {
	testParse(t, `
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
	testParse(t, `
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

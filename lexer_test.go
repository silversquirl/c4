package main

import (
	"strings"
	"testing"
)

func testTokens(t *testing.T, src string, toks []Token) {
	tokC := make(chan Token)
	go Tokenize(src, tokC)
	for _, tok := range toks {
		tok2 := <-tokC
		if tok != tok2 {
			t.Fatalf("Tokens do not match: expected %v, got %v", tok, tok2)
		}
	}
	if (<-tokC).Ty != TEOF {
		t.Fatal("Too many tokens")
	}
}

func TestTokenize(t *testing.T) {
	testTokens(t, `
		// Comment
		;,()[]{} \
		=+-*/%!|^&<> \
		<<>>&&||==!=<=>= \
		else extern fn for if pub return type var
		elseexternfnforifpubreturntypevar \
		fooBar _ _foo foo_ \
		FooBar \
		"" "hello" 0 1 -1 0. .0 0.0 1.1 -1.1 \
	`, []Token{
		{TSemi, ";"}, {TComma, ","}, {TLParen, "("}, {TRParen, ")"},
		{TLSquare, "["}, {TRSquare, "]"}, {TLBrace, "{"}, {TRBrace, "}"},

		{TEquals, "="}, {TPlus, "+"}, {TMinus, "-"}, {TAster, "*"},
		{TSlash, "/"}, {TPerc, "%"}, {TExcl, "!"}, {TPipe, "|"},
		{TCaret, "^"}, {TAmp, "&"}, {TLess, "<"}, {TGreater, ">"},

		{TShl, "<<"}, {TShr, ">>"}, {TLand, "&&"}, {TLor, "||"},
		{TCeq, "=="}, {TCne, "!="}, {TCle, "<="}, {TCge, ">="},

		{TKelse, "else"}, {TKextern, "extern"}, {TKfn, "fn"}, {TKfor, "for"},
		{TKif, "if"}, {TKpub, "pub"}, {TKreturn, "return"}, {TKtype, "type"},
		{TKvar, "var"},

		{TIdent, "elseexternfnforifpubreturntypevar"}, {TIdent, "fooBar"}, {TIdent, "_"}, {TIdent, "_foo"},
		{TIdent, "foo_"}, {TType, "FooBar"},

		{TString, `""`}, {TString, `"hello"`}, {TInteger, "0"}, {TInteger, "1"},
		{TInteger, "-1"}, {TFloat, "0."}, {TFloat, ".0"}, {TFloat, "0.0"},
		{TFloat, "1.1"}, {TFloat, "-1.1"},
	})
}

func TestAutoSemi(t *testing.T) {
	testTokens(t, strings.Join([]string{
		// Non-auto-semi tokens
		`;`, `,`,
		`(`, `[`, `{`,

		`=`, `+`, `-`, `*`,
		`/`, `%`, `!`, `|`,
		`^`, `&`, `<`, `>`,

		`<<`, `>>`, `&&`, `||`,
		`==`, `!=`, `<=`, `>=`,

		`else`, `extern`, `fn`, `for`,
		`if`, `pub`, `return`, `type`,
		`var`,

		// Auto-semi tokens
		`)`, `]`, `}`,
		`foo`, `Foo`,
		`""`, `0`, `0.`,
		``,
	}, "\n"), []Token{
		// Non-auto-semi tokens
		{TSemi, ";"}, {TComma, ","},
		{TLParen, "("}, {TLSquare, "["}, {TLBrace, "{"},

		{TEquals, "="}, {TPlus, "+"}, {TMinus, "-"}, {TAster, "*"},
		{TSlash, "/"}, {TPerc, "%"}, {TExcl, "!"}, {TPipe, "|"},
		{TCaret, "^"}, {TAmp, "&"}, {TLess, "<"}, {TGreater, ">"},

		{TShl, "<<"}, {TShr, ">>"}, {TLand, "&&"}, {TLor, "||"},
		{TCeq, "=="}, {TCne, "!="}, {TCle, "<="}, {TCge, ">="},

		{TKelse, "else"}, {TKextern, "extern"}, {TKfn, "fn"}, {TKfor, "for"},
		{TKif, "if"}, {TKpub, "pub"}, {TKreturn, "return"}, {TKtype, "type"},
		{TKvar, "var"},

		// Non-auto-semi tokens
		{TRParen, ")"}, {TSemi, "\n"}, {TRSquare, "]"}, {TSemi, "\n"},
		{TRBrace, "}"}, {TSemi, "\n"}, {TIdent, "foo"}, {TSemi, "\n"},
		{TType, "Foo"}, {TSemi, "\n"}, {TString, `""`}, {TSemi, "\n"},
		{TInteger, "0"}, {TSemi, "\n"}, {TFloat, "0."}, {TSemi, "\n"},
	})
}

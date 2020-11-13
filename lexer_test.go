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
			t.Errorf("Tokens do not match: expected %v, got %v", tok, tok2)
		}
	}
	if (<-tokC).Ty != TEOF {
		t.Error("Too many tokens")
	}
}

func TestTokenize(t *testing.T) {
	testTokens(t, `
		// Comment
		;,()[]{} \
		+= -= *= /= %= |= ^= \
		&= <<= >>= &&= ||= \
		=+-*/%!|^&<> \
		<<>>&&||<=>===!= \
		else extern fn for if pub return type var
		elseexternfnforifpubreturntypevar \
		fooBar _ _foo foo_ \
		FooBar \
		"" "hello" 'a' 0 1 0. .0 0.0 1.1 -1.1 \
	`, []Token{
		{16, TSemi, ";"}, {17, TComma, ","}, {18, TLParen, "("}, {19, TRParen, ")"},
		{20, TLSquare, "["}, {21, TRSquare, "]"}, {22, TLBrace, "{"}, {23, TRBrace, "}"},

		{29, TMadd, "+="}, {32, TMsub, "-="}, {35, TMmul, "*="}, {38, TMdiv, "/="},
		{41, TMmod, "%="}, {44, TMor, "|="}, {47, TMxor, "^="}, {54, TMand, "&="},
		{57, TMshl, "<<="}, {61, TMshr, ">>="}, {65, TMland, "&&="}, {69, TMlor, "||="},

		{77, TEquals, "="}, {78, TPlus, "+"}, {79, TMinus, "-"}, {80, TAster, "*"},
		{81, TSlash, "/"}, {82, TPerc, "%"}, {83, TExcl, "!"}, {84, TPipe, "|"},
		{85, TCaret, "^"}, {86, TAmp, "&"}, {87, TLess, "<"}, {88, TGreater, ">"},

		{94, TShl, "<<"}, {96, TShr, ">>"}, {98, TLand, "&&"}, {100, TLor, "||"},
		{102, TCle, "<="}, {104, TCge, ">="}, {106, TCeq, "=="}, {108, TCne, "!="},

		{115, TKelse, "else"}, {120, TKextern, "extern"}, {127, TKfn, "fn"}, {130, TKfor, "for"},
		{134, TKif, "if"}, {137, TKpub, "pub"}, {141, TKreturn, "return"}, {148, TKtype, "type"},
		{153, TKvar, "var"},

		{159, TIdent, "elseexternfnforifpubreturntypevar"}, {197, TIdent, "fooBar"}, {204, TIdent, "_"}, {206, TIdent, "_foo"},
		{211, TIdent, "foo_"}, {220, TType, "FooBar"},

		{231, TString, ""}, {234, TString, "hello"}, {242, TRune, "a"}, {246, TInteger, "0"},
		{248, TInteger, "1"}, {250, TFloat, "0."}, {253, TFloat, ".0"}, {256, TFloat, "0.0"},
		{260, TFloat, "1.1"}, {264, TFloat, "-1.1"},
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
		`if`, `pub`, `type`, `var`,

		// Auto-semi tokens
		`)`, `]`, `}`,
		`foo`, `Foo`,
		`""`, `'a'`, `0`, `0.`,
		`return`,
		``,
	}, "\n"), []Token{
		// Non-auto-semi tokens
		{0, TSemi, ";"}, {2, TComma, ","},
		{4, TLParen, "("}, {6, TLSquare, "["}, {8, TLBrace, "{"},

		{10, TEquals, "="}, {12, TPlus, "+"}, {14, TMinus, "-"}, {16, TAster, "*"},
		{18, TSlash, "/"}, {20, TPerc, "%"}, {22, TExcl, "!"}, {24, TPipe, "|"},
		{26, TCaret, "^"}, {28, TAmp, "&"}, {30, TLess, "<"}, {32, TGreater, ">"},

		{34, TShl, "<<"}, {37, TShr, ">>"}, {40, TLand, "&&"}, {43, TLor, "||"},
		{46, TCeq, "=="}, {49, TCne, "!="}, {52, TCle, "<="}, {55, TCge, ">="},

		{58, TKelse, "else"}, {63, TKextern, "extern"}, {70, TKfn, "fn"}, {73, TKfor, "for"},
		{77, TKif, "if"}, {80, TKpub, "pub"}, {84, TKtype, "type"}, {89, TKvar, "var"},

		// Non-auto-semi tokens
		{93, TRParen, ")"}, {94, TSemi, "\n"}, {95, TRSquare, "]"}, {96, TSemi, "\n"},
		{97, TRBrace, "}"}, {98, TSemi, "\n"}, {99, TIdent, "foo"}, {102, TSemi, "\n"},
		{103, TType, "Foo"}, {106, TSemi, "\n"}, {107, TString, ""}, {109, TSemi, "\n"},
		{110, TRune, "a"}, {113, TSemi, "\n"}, {114, TInteger, "0"}, {115, TSemi, "\n"},
		{116, TFloat, "0."}, {118, TSemi, "\n"}, {119, TKreturn, "return"}, {125, TSemi, "\n"},
	})
}

//go:generate stringer -type TokenType -linecomment
package main

import (
	"regexp"
	"strings"
)

func Tokenize(code string, toks chan<- Token) {
	l := lexer{toks: toks}
	l.tokenize(code)
}

type lexer struct {
	toks chan<- Token
	tok  Token
}

func (l *lexer) tokenize(code string) {
	for _, m := range lexerRegex.FindAllStringSubmatch(code, -1) {
		m = m[1:]
		for i, rule := range lexerRules {
			if m[i] != "" {
				tok := Token{rule.Ty, m[i]}
				if rule.Sub != nil {
					tok = rule.Sub(tok)
				}
				l.emitToken(tok)
				break
			}
		}
	}
	l.flush()
	close(l.toks)
}
func (l *lexer) emitToken(tok Token) {
	switch tok.Ty {
	case TComment, TSpace:
		return
	case TNewline:
		if l.tok.Ty.autoSemi() {
			tok.Ty = TSemi
		} else {
			return
		}
	}

	l.flush()
	l.tok = tok
}
func (l *lexer) flush() {
	switch l.tok.Ty {
	case 0, TBackslash:
	default:
		l.toks <- l.tok
	}
}

func parseKeyword(tok Token) Token {
	if ty, ok := lexerKw[tok.S]; ok {
		tok.Ty = ty
	}
	return tok
}
func parseString(tok Token) Token {
	tok.S = tok.S[1 : len(tok.S)-1]
	return tok
}
func parseInt(tok Token) Token {
	return tok
}
func parseFloat(tok Token) Token {
	return tok
}

var lexerRegex *regexp.Regexp
var lexerRules []lexerRule
var lexerKw = map[string]TokenType{}

type lexerRule struct {
	Ty  TokenType
	Sub func(Token) Token
}

func init() {
	// Generate lexer regex and match mappings
	// Wouldn't it be nice if we could do this at compile time?
	regexBuilder := &strings.Builder{}
	for ty := TEOF; ty < LexTokenMax; ty++ {
		var pat string            // regex pattern
		var sub func(Token) Token // sub-lexer

		switch ty {
		case TEOF:
			continue

		case TComment:
			pat = `//.*`
		case TSpace:
			pat = `[ \t]+`
		case TNewline:
			pat = `\n`
		case TIdent:
			pat = `[\p{Ll}_][\pL\pN]*_*`
			sub = parseKeyword
		case TType:
			pat = `\p{Lu}[\pL\pN]*`
		case TString:
			pat = `"(?:[^"]|\\.)*"`
			sub = parseString
		case TInteger:
			// TODO: type suffixes
			pat = `[-+]?(?:0x[0-9A-Fa-f_]+|0b[01_]+|0[0-7_]*|[0-9_]+)`
			sub = parseInt
		case TFloat:
			// TODO: type suffixes, hex floats(?), exponents
			pat = `[-+]?(?:\d+\.\d*|\.\d+)`
			sub = parseFloat

		case TInvalid:
			pat = `.`

		default:
			// Exploit the generated String() method to autogenerate rules for operators
			s := ty.String()
			e := len(s) - 1
			if s[0] == '\'' && s[e] == '\'' {
				pat = regexp.QuoteMeta(s[1:e])
			} else {
				panic("Token with no lexer rule: " + s)
			}
		}

		// Validate pattern - all should be fine unless I messed up
		if pat == "" {
			panic("Pattern for " + ty.String() + " is empty")
		}
		patRe, err := regexp.Compile(pat)
		if err != nil {
			panic("Pattern for " + ty.String() + " failed to compile: " + err.Error())
		}
		if patRe.NumSubexp() > 0 {
			panic("Pattern for " + ty.String() + " contains subexpressions")
		}

		// Add rule
		if len(lexerRules) > 0 {
			regexBuilder.WriteByte('|')
		}
		regexBuilder.WriteByte('(')
		regexBuilder.WriteString(pat)
		regexBuilder.WriteByte(')')
		lexerRules = append(lexerRules, lexerRule{ty, sub})
	}
	lexerRegex = regexp.MustCompile(regexBuilder.String())

	// Generate keyword table
	for ty := TKeywordStart + 1; ty < TKeywordEnd; ty++ {
		kw := ty.String()
		kw = kw[1 : len(kw)-1]
		lexerKw[kw] = ty
	}
}

const (
	TEOF TokenType = iota // end of file

	TComment // comment
	TSpace   // whitespace

	TNewline   // newline
	TBackslash // '\'
	TSemi      // ';'
	TComma     // ','

	// Matching pairs
	TLParen  // '('
	TRParen  // ')'
	TLSquare // '['
	TRSquare // ']'
	TLBrace  // '{'
	TRBrace  // '}'

	// Identifiers
	TIdent // identifier
	TType  // type name

	// Constants
	TString  // string literal
	TFloat   // float literal
	TInteger // integer literal

	// Mutation operators
	TMadd  // '+='
	TMsub  // '-='
	TMmul  // '*='
	TMdiv  // '/='
	TMmod  // '%='
	TMor   // '|='
	TMxor  // '^='
	TMand  // '&='
	TMshl  // '<<='
	TMshr  // '>>='
	TMland // '&&='
	TMlor  // '||='

	// Multi-char operators
	TShl  // '<<'
	TShr  // '>>'
	TLand // '&&'
	TLor  // '||'
	TCeq  // '=='
	TCne  // '!='
	TCle  // '<='
	TCge  // '>='

	// Single character operators
	TEquals  // '='
	TPlus    // '+'
	TMinus   // '-'
	TAster   // '*'
	TSlash   // '/'
	TPerc    // '%'
	TExcl    // '!'
	TPipe    // '|'
	TCaret   // '^'
	TAmp     // '&'
	TLess    // '<'
	TGreater // '>'
	TDot     // '.'

	TInvalid // invalid token

	// Tokens after this point will not be produced by the 1st lexer layer
	// They may be produced by sublexers
	LexTokenMax

	// Keywords
	TKeywordStart
	TKelse     // 'else'
	TKextern   // 'extern'
	TKfn       // 'fn'
	TKfor      // 'for'
	TKif       // 'if'
	TKpub      // 'pub'
	TKreturn   // 'return'
	TKstruct   // 'struct'
	TKtype     // 'type'
	TKunion    // 'union'
	TKvar      // 'var'
	TKvariadic // 'variadic'
	TKeywordEnd
)

type Token struct {
	Ty TokenType
	S  string
}

type TokenType int

// autoSemi returns true if subsequent newlines should be replaced with semicolons
func (ty TokenType) autoSemi() bool {
	switch ty {
	case TRParen, TRSquare, TRBrace:
	case TIdent, TType:
	case TString, TInteger, TFloat:
	default:
		return false
	}
	return true
}

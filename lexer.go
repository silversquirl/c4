//go:generate stringer -type TokenType -linecomment
package main

import (
	"io"
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
			text := m[i]
			if text != "" {
				if rule.Sub != nil {
					text = rule.Sub(text)
				}
				l.emitToken(text, rule.Ty)
				break
			}
		}
	}
	l.finish()
	close(l.toks)
}
func (l *lexer) emitToken(text string, ty TokenType) {
	switch ty {
	case TComment, TSpace:
		return
	case TNewline:
		if l.tok.Ty.autoSemi() {
			ty = TSemi
		} else {
			return
		}
	}

	if l.tok.Ty > 0 {
		l.toks <- l.tok
	}

	l.tok = Token{ty, text}
}
func (l *lexer) finish() {
	if l.tok.Ty > 0 {
		l.toks <- l.tok
	}
}

func parseString(text string) string {
	return text
}
func parseInt(text string) string {
	return text
}
func parseFloat(text string) string {
	return text
}

var lexerRegex *regexp.Regexp
var lexerRules []lexerRule

type lexerRule struct {
	Ty  TokenType
	Sub func(string) string
}

func init() {
	// Generate lexer regex and match mappings
	// Wouldn't it be nice if we could do this at compile time?
	regexBuilder := &strings.Builder{}
	for ty := TEOF; ty < TokenTypeMax; ty++ {
		var pat string              // regex pattern
		var sub func(string) string // sub-lexer

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
		case TType:
			pat = `\p{Lu}[\pL\pN]*`
		case TString:
			pat = `"(?:[^"]|\\")*"`
			sub = parseString
		case TInteger:
			// TODO: type suffixes
			pat = `0x[0-9A-Fa-f_]+|0b[01_]+|0[0-7_]*|[0-9_]+`
			sub = parseInt
		case TFloat:
			// TODO: type suffixes, hex floats(?), exponents
			pat = `\d+\.\d*|\.\d+`
			sub = parseFloat

		case TInvalid:
			pat = `.`

		default:
			// Exploit the generated String() method to autogenerate rules for operators and keywords
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
}

type cachingReader struct {
	strings.Builder
	r io.RuneReader
}

func (r *cachingReader) ReadRune() (ch rune, size int, err error) {
	ch, size, err = r.r.ReadRune()
	if err == nil {
		r.Builder.WriteRune(ch)
	}
	return
}

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

const (
	TEOF TokenType = iota // end of file

	TComment // comment
	TSpace   // whitespace

	TNewline // newline
	TSemi    // ';'
	TComma   // ','

	// Matching pairs
	TLParen  // '('
	TRParen  // ')'
	TLSquare // '['
	TRSquare // ']'
	TLBrace  // '{'
	TRBrace  // '}'

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

	// Multi-char operators
	TShl  // '<<'
	TShr  // '>>'
	TLand // '&&'
	TLor  // '||'
	TCeq  // '=='
	TCne  // '!='
	TCle  // '<='
	TCge  // '>='

	// Keywords
	TKelse   // 'else'
	TKextern // 'extern'
	TKfn     // 'fn'
	TKfor    // 'for'
	TKif     // 'if'
	TKpub    // 'pub'
	TKreturn // 'return'
	TKtype   // 'type'
	TKvar    // 'var'

	// Identifiers
	TIdent // identifier
	TType  // type name

	// Constants
	TString  // string literal
	TInteger // integer literal
	TFloat   // float literal

	TInvalid // invalid token

	TokenTypeMax
	TokenTypeMin   = TEOF
	TokenTypeKwMin = TKelse
	TokenTypeKwMax = TKvar + 1
)

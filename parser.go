package main

import (
	"fmt"
	"strings"
)

func init() {
}

func Parse(code string) (prog Program, err error) {
	toks := make(chan Token)
	go Tokenize(code, toks)
	p := parser{<-toks, toks}
	defer func() {
		switch e := recover().(type) {
		case nil:
		case string:
			line := 1
			for i := 0; i < p.tok.Off; i++ {
				if code[i] == '\n' {
					line++
				}
			}
			err = fmt.Errorf("Parse error at line %d: %s", line, e)
		default:
			panic(e)
		}
	}()
	return p.parseProgram(), nil
}

type parser struct {
	tok  Token
	toks <-chan Token
}

func (p *parser) errExpect(what string) {
	panic("Expected " + what + "; got " + p.peek().String())
}

func (p *parser) peek() TokenType {
	return p.tok.Ty
}
func (p *parser) next() Token {
	tok := p.tok
	p.tok = <-p.toks
	return tok
}

func (p *parser) accept(types ...TokenType) (ok bool) {
	ty := p.peek()
	for _, arg := range types {
		if ty == arg {
			p.next()
			return true
		}
	}
	return false
}
func (p *parser) require(types ...TokenType) Token {
	tok := p.tok
	if p.accept(types...) {
		return tok
	}

	var what string
	if len(types) == 1 {
		what = types[0].String()
	} else {
		b := &strings.Builder{}
		b.WriteString("one of")
		for i, ty := range types {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteByte(' ')
			b.WriteString(ty.String())
		}
		what = b.String()
	}
	p.errExpect(what)
	panic("unreachable")
}

type listParser struct {
	p        *parser
	started  bool
	sep, end TokenType
}

func (p *listParser) next() bool {
	if !p.started {
		p.started = true
		return !p.p.accept(p.end)
	} else {
		done := p.p.require(p.end, p.sep).Ty == p.end
		if !done {
			done = p.p.accept(p.end)
		}
		return !done
	}
}
func (p *parser) list(sep, end TokenType) listParser {
	return listParser{p, false, sep, end}
}

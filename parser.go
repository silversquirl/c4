package main

import (
	"errors"
	"strings"
)

func init() {
}

type toplevelParselet func(*parser, Token, bool) []Toplevel
type statementParselet func(*parser, Token) Statement
type prefixExprParselet struct {
	prec int
	fun  func(int, *parser, Token) Expression
}
type exprParselet struct {
	prec int
	fun  func(int, *parser, Token, Expression) Expression
}
type typeParselet func(*parser, Token) TypeExpr

func Parse(code string) (prog Program, err error) {
	toks := make(chan Token)
	go Tokenize(code, toks)
	p := parser{<-toks, toks}
	defer func() {
		switch e := recover().(type) {
		case nil:
		case string:
			err = errors.New(e)
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

	msg := &strings.Builder{}
	msg.WriteString("Invalid token: expected ")
	if len(types) == 1 {
		msg.WriteString(types[0].String())
	} else {
		msg.WriteString("one of")
		for i, ty := range types {
			if i > 0 {
				msg.WriteByte(',')
			}
			msg.WriteByte(' ')
			msg.WriteString(ty.String())
		}
	}
	msg.WriteString("; got ")
	msg.WriteString(tok.Ty.String())
	panic(msg.String())
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

func (p *parser) parseProgram() (prog Program) {
	for p.peek() != TEOF {
		prog = append(prog, p.parseToplevel()...)
	}
	return prog
}

func (p *parser) parseToplevel() []Toplevel {
	// TODO: more modifiers
	pub := false
	if p.peek() == TKpub {
		pub = true
		p.next()
	}

	tok := p.next()
	pl, ok := toplevelParselets[tok.Ty]
	if !ok {
		panic("Expected toplevel construct, got " + tok.Ty.String())
	}
	return pl(p, tok, pub)
}

func (p *parser) parseStatement() Statement {
	pl, ok := statementParselets[p.peek()]
	if ok {
		return pl(p, p.next())
	} else {
		return ExprStmt{p.parseExpression(0)}
	}
}

func (p *parser) parseExpression(prec int) Expression {
	tok := p.next()
	pl, ok := prefixExprParselets[tok.Ty]
	if !ok {
		panic("Expected expression, got " + tok.Ty.String())
	}
	left := pl.fun(pl.prec, p, tok)

	for {
		pl := exprParselets[p.peek()]
		if pl.prec <= prec {
			return left
		}
		left = pl.fun(pl.prec, p, p.next(), left)
	}
}

func (p *parser) parseVarTypes() []VarDecl {
	names := []string{p.require(TIdent).S}
	for {
		if !p.accept(TComma) {
			break
		}
		if p.peek() != TIdent {
			break
		}
		names = append(names, p.next().S)
	}
	ty := p.parseType()

	decls := make([]VarDecl, len(names))
	for i, name := range names {
		decls[i] = VarDecl{name, ty}
	}
	return decls
}

func (p *parser) canParseType() bool {
	_, ok := typeParselets[p.peek()]
	return ok
}
func (p *parser) parseType() TypeExpr {
	tok := p.next()
	pl, ok := typeParselets[tok.Ty]
	if !ok {
		panic("Expected type, got " + tok.Ty.String())
	}
	return pl(p, tok)
}

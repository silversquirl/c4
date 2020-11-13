package main

type toplevelParselet func(*parser, Token) Toplevel
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

var toplevelParselets map[TokenType]toplevelParselet
var statementParselets map[TokenType]statementParselet
var prefixExprParselets map[TokenType]prefixExprParselet
var exprParselets map[TokenType]exprParselet
var typeParselets map[TokenType]typeParselet

func (p *parser) parseProgram() (prog Program) {
	for p.peek() != TEOF {
		prog = append(prog, p.parseToplevel())
		p.require(TSemi, TEOF)
	}
	return prog
}

func (p *parser) parseToplevel() Toplevel {
	pl, ok := toplevelParselets[p.peek()]
	if !ok {
		p.errExpect("toplevel construct")
	}
	return pl(p, p.next())
}

func init() {
	toplevelParselets = map[TokenType]toplevelParselet{
		TKns: func(p *parser, tok Token) Toplevel {
			ns := NamespaceTL{Name: p.require(TIdent).S}
			p.require(TLBrace)
			for l := p.list(TSemi, TRBrace); l.next(); {
				ns.Body = append(ns.Body, p.parseToplevel())
			}
			return ns
		},

		TKpub: func(p *parser, tok Token) Toplevel {
			switch tl := p.parseToplevel().(type) {
			case Function:
				tl.Pub = true
				return tl
			}
			panic("Expected function")
		},
		TKvariadic: func(p *parser, tok Token) Toplevel {
			switch tl := p.parseToplevel().(type) {
			case VarsDecl:
				if ty, ok := tl.Ty.(FuncTypeExpr); ok {
					ty.Var = true
					tl.Ty = ty
					return tl
				}
			}
			panic("Expected function declaration")
		},

		TKfn: func(p *parser, tok Token) Toplevel {
			// Parse function signature
			name := p.require(TIdent).S
			p.require(TLParen)
			var params []VarDecl
			for l := p.list(TComma, TRParen); l.next(); {
				params = append(params, p.parseVarTypes().Decls()...)
			}
			ret := p.parseType()

			if p.peek() == TLBrace {
				// Parse function body
				return Function{false, name, params, ret, p.parseBlock()}
			} else {
				// No body, just a declaration
				paramTy := make([]TypeExpr, len(params))
				for i, param := range params {
					paramTy[i] = param.Ty
				}
				return VarsDecl{[]string{name}, FuncTypeExpr{false, paramTy, ret}}
			}
		},

		TKvar: func(p *parser, tok Token) Toplevel {
			return p.parseVarTypes()
		},

		TKtype: func(p *parser, tok Token) Toplevel {
			name := p.require(TType).S
			if p.accept(TEquals) {
				return TypeAlias{name, p.parseType()}
			} else {
				return TypeDef{name, p.parseType()}
			}
		},
	}
}

func (p *parser) parseBlock() (stmts []Statement) {
	p.require(TLBrace)
	for l := p.list(TSemi, TRBrace); l.next(); {
		stmts = append(stmts, p.parseStatement())
	}
	return
}

func (p *parser) parseStatement() Statement {
	pl, ok := statementParselets[p.peek()]
	if ok {
		return pl(p, p.next())
	} else {
		return ExprStmt{p.parseExpression(0)}
	}
}

func init() {
	statementParselets = map[TokenType]statementParselet{
		TKreturn: func(p *parser, tok Token) Statement {
			e := p.parseExpression(0)
			return ReturnStmt{e}
		},
		TKvar: func(p *parser, tok Token) Statement {
			return p.parseVarTypes()
		},

		TKif: func(p *parser, tok Token) Statement {
			i := IfStmt{}
			i.Cond = p.parseExpression(0)
			i.Then = p.parseBlock()

			if p.accept(TKelse) {
				if p.peek() == TKif {
					i.Else = []Statement{p.parseStatement()}
				} else {
					i.Else = p.parseBlock()
				}
			}
			return i
		},

		TKfor: func(p *parser, tok Token) Statement {
			if p.peek() == TLBrace {
				// No arguments
				return ForStmt{nil, nil, nil, p.parseBlock()}
			}

			var init Statement
			var cond, step Expression
			if !p.accept(TSemi) {
				init = p.parseStatement()
				if !p.accept(TSemi) {
					// One arg
					if cond, ok := init.(ExprStmt); !ok {
						panic("Expected expression, got statement")
					} else {
						return ForStmt{nil, cond, nil, p.parseBlock()}
					}
				}
			}

			if !p.accept(TSemi) {
				cond = p.parseExpression(0)
				p.require(TSemi)
			}
			if p.peek() != TLBrace {
				step = p.parseExpression(0)
			}
			return ForStmt{init, cond, step, p.parseBlock()}
		},
	}
}

func (p *parser) parseExpression(prec int) Expression {
	pl, ok := prefixExprParselets[p.peek()]
	if !ok {
		p.errExpect("expression")
	}
	left := pl.fun(pl.prec, p, p.next())

	for {
		pl := exprParselets[p.peek()]
		if pl.prec <= prec {
			return left
		}
		left = pl.fun(pl.prec, p, p.next(), left)
	}
}

func init() {
	prefOpMap := map[string]PrefixOperator{}
	for op := PrefixOperator(1); op < PrefixOperatorMax; op++ {
		prefOpMap[op.String()] = op
	}

	prefix := func(prec int, p *parser, tok Token) Expression {
		op, ok := prefOpMap[tok.S]
		if !ok {
			panic("Invalid prefix operator: " + tok.S)
		}

		v := p.parseExpression(prec)
		return PrefixExpr{op, v}
	}

	prefixExprParselets = map[TokenType]prefixExprParselet{
		TIdent: {PrecLiteral, func(prec int, p *parser, tok Token) Expression {
			return VarExpr(tok.S)
		}},
		TString: {PrecLiteral, func(prec int, p *parser, tok Token) Expression {
			return StringExpr(tok.S)
		}},
		TRune: {PrecLiteral, func(prec int, p *parser, tok Token) Expression {
			return RuneExpr([]rune(tok.S)[0])
		}},
		TFloat: {PrecLiteral, func(prec int, p *parser, tok Token) Expression {
			return FloatExpr(tok.S)
		}},
		TInteger: {PrecLiteral, func(prec int, p *parser, tok Token) Expression {
			return IntegerExpr(tok.S)
		}},

		TLParen: {PrecGroup, func(prec int, p *parser, tok Token) Expression {
			e := p.parseExpression(0)
			p.require(TRParen)
			return e
		}},
		TLSquare: {PrecGroup, func(prec int, p *parser, tok Token) Expression {
			e := p.parseExpression(0)
			p.require(TRSquare)
			return DerefExpr{e}
		}},

		TAmp: {PrecPrefix, func(prec int, p *parser, tok Token) Expression {
			v, ok := p.parseExpression(prec).(LValue)
			if !ok {
				panic("Reference of non-lvalue")
			}
			return RefExpr{v}
		}},

		TExcl:  {PrecPrefix, prefix},
		TCaret: {PrecPrefix, prefix},
		TMinus: {PrecPrefix, prefix},
		TPlus:  {PrecPrefix, prefix},
	}
}

func init() {
	binOpMap := map[string]BinaryOperator{}
	for op := BinaryOperator(1); op < BinaryOperatorMax; op++ {
		binOpMap[op.String()] = op
	}

	binary := func(prec int, p *parser, tok Token, left Expression) Expression {
		if op, ok := binOpMap[tok.S]; !ok {
			panic("Invalid binary operator: " + tok.S)
		} else {
			return BinaryExpr{op, left, p.parseExpression(prec)}
		}
	}

	boolean := func(prec int, p *parser, tok Token, left Expression) Expression {
		var op BooleanOperator
		switch tok.S {
		case "&&":
			op = BoolAnd
		case "||":
			op = BoolOr
		}
		return BooleanExpr{op, left, p.parseExpression(prec)}
	}

	mutate := func(prec int, p *parser, tok Token, left Expression) Expression {
		binOp := tok.S[:len(tok.S)-1]
		if l, ok := left.(LValue); !ok {
			panic("Mutate of non-lvalue")
		} else if op, ok := binOpMap[binOp]; !ok {
			panic("Invalid mutation operator: " + tok.S)
		} else {
			return MutateExpr{op, l, p.parseExpression(prec - 1)}
		}
	}

	exprParselets = map[TokenType]exprParselet{
		TDot: {PrecAccess, func(prec int, p *parser, tok Token, left Expression) Expression {
			if l, ok := left.(LValue); !ok {
				panic("Field access of non-lvalue")
			} else {
				return AccessExpr{l, p.require(TIdent).S}
			}
		}},

		TEquals: {PrecAssign, func(prec int, p *parser, tok Token, left Expression) Expression {
			if l, ok := left.(LValue); !ok {
				panic("Assign to non-lvalue")
			} else {
				return AssignExpr{l, p.parseExpression(prec - 1)}
			}
		}},

		TPlus:  {PrecSum, binary},
		TMinus: {PrecSum, binary},
		TAster: {PrecMul, binary},
		TSlash: {PrecMul, binary},
		TPerc:  {PrecMul, binary},

		TPipe:  {PrecBitwise, binary},
		TCaret: {PrecBitwise, binary},
		TAmp:   {PrecBitwise, binary},
		TShl:   {PrecShift, binary},
		TShr:   {PrecShift, binary},

		TCeq:     {PrecCompare, binary},
		TCne:     {PrecCompare, binary},
		TLess:    {PrecCompare, binary},
		TGreater: {PrecCompare, binary},
		TCle:     {PrecCompare, binary},
		TCge:     {PrecCompare, binary},

		TLand: {PrecLand, boolean},
		TLor:  {PrecLor, boolean},

		TMadd:  {PrecAssign, mutate},
		TMsub:  {PrecAssign, mutate},
		TMmul:  {PrecAssign, mutate},
		TMdiv:  {PrecAssign, mutate},
		TMmod:  {PrecAssign, mutate},
		TMor:   {PrecAssign, mutate},
		TMxor:  {PrecAssign, mutate},
		TMand:  {PrecAssign, mutate},
		TMshl:  {PrecAssign, mutate},
		TMshr:  {PrecAssign, mutate},
		TMland: {PrecAssign, mutate},
		TMlor:  {PrecAssign, mutate},

		TLParen: {PrecCall, func(prec int, p *parser, tok Token, left Expression) Expression {
			call := CallExpr{Func: left}
			for l := p.list(TComma, TRParen); l.next(); {
				call.Args = append(call.Args, p.parseExpression(0))
			}
			return call
		}},
	}
}

func (p *parser) parseVarTypes() (d VarsDecl) {
	for {
		d.Names = append(d.Names, p.require(TIdent).S)
		if !p.accept(TComma) {
			break
		}
	}

	d.Ty = p.parseType()
	if d.Ty == nil {
		p.errExpect("type")
	}
	return
}

func (p *parser) parseType() TypeExpr {
	pl, ok := typeParselets[p.peek()]
	if !ok {
		return nil
	}
	return pl(p, p.next())
}

func init() {
	composite := func(p *parser) (fields []VarDecl) {
		p.require(TLBrace)
		for l := p.list(TSemi, TRBrace); l.next(); {
			d := p.parseVarTypes()
			fields = append(fields, d.Decls()...)
		}
		return
	}

	typeParselets = map[TokenType]typeParselet{
		TType: func(p *parser, tok Token) TypeExpr {
			return NamedTypeExpr(tok.S)
		},

		TLSquare: func(p *parser, tok Token) TypeExpr {
			to := p.parseType()
			if to == nil {
				p.errExpect("type")
			}
			p.require(TRSquare)
			return PointerTypeExpr{to}
		},

		TKfn: func(p *parser, tok Token) TypeExpr {
			t := FuncTypeExpr{}
			p.require(TLParen)
			for l := p.list(TComma, TRParen); l.next(); {
				// FIXME: doesn't work with multiple params per type
				p.accept(TIdent)
				ty := p.parseType()
				if ty == nil {
					p.errExpect("type")
				}
				t.Param = append(t.Param, ty)
			}
			t.Ret = p.parseType()
			return PointerTypeExpr{t}
		},

		TKstruct: func(p *parser, tok Token) TypeExpr {
			return StructTypeExpr(composite(p))
		},
		TKunion: func(p *parser, tok Token) TypeExpr {
			return UnionTypeExpr(composite(p))
		},
	}
}

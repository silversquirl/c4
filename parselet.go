package main

var toplevelParselets = map[TokenType]toplevelParselet{
	TKfn: func(p *parser, tok Token, pub bool) []Toplevel {
		// Parse function signature
		name := p.require(TIdent).S
		p.require(TLParen)
		var params []VarDecl
		for l := p.list(TComma, TRParen); l.next(); {
			params = append(params, p.parseVarTypes()...)
		}
		ret := p.parseType()

		var tl Toplevel
		if p.accept(TLBrace) {
			// Parse function body
			var body []Statement
			for l := p.list(TSemi, TRBrace); l.next(); {
				body = append(body, p.parseStatement()...)
			}
			tl = Function{pub, name, params, ret, body}
		} else {
			// No body, just a declaration
			paramTy := make([]TypeExpr, len(params))
			for i, param := range params {
				paramTy[i] = param.Ty
			}
			tl = VarDecl{name, FuncTypeExpr{paramTy, ret}}
		}
		return []Toplevel{tl}
	},
	TKvar: func(p *parser, tok Token, pub bool) []Toplevel {
		vds := p.parseVarTypes()
		tls := make([]Toplevel, len(vds))
		for i, vd := range vds {
			tls[i] = vd
		}
		return tls
	},
}

var statementParselets = map[TokenType]statementParselet{
	TKreturn: func(p *parser, tok Token) []Statement {
		e := p.parseExpression(0)
		return []Statement{ReturnStmt{e}}
	},
	TKvar: func(p *parser, tok Token) []Statement {
		vds := p.parseVarTypes()
		stmts := make([]Statement, len(vds))
		for i, vd := range vds {
			stmts[i] = vd
		}
		return stmts
	},
}

var prefixExprParselets map[TokenType]prefixExprParselet

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

var exprParselets map[TokenType]exprParselet

func init() {
	binOpMap := map[string]BinaryOperator{}
	for op := BinaryOperator(1); op < BinaryOperatorMax; op++ {
		binOpMap[op.String()] = op
	}

	binary := func(prec int, p *parser, tok Token, left Expression) Expression {
		op, ok := binOpMap[tok.S]
		if !ok {
			panic("Invalid binary operator: " + tok.S)
		}

		right := p.parseExpression(prec)
		return BinaryExpr{op, left, right}
	}

	exprParselets = map[TokenType]exprParselet{
		TEquals: {PrecAssign, func(prec int, p *parser, tok Token, left Expression) Expression {
			l, ok := left.(LValue)
			if !ok {
				panic("Assign to non-lvalue")
			}
			return AssignExpr{l, p.parseExpression(prec - 1)}
		}},

		TPlus:  {PrecSum, binary},
		TMinus: {PrecSum, binary},
		TAster: {PrecMul, binary},
		TSlash: {PrecMul, binary},
		TPerc:  {PrecMul, binary},

		TPipe:  {PrecOr, binary},
		TCaret: {PrecXor, binary},
		TAmp:   {PrecAnd, binary},
		TShl:   {PrecShift, binary},
		TShr:   {PrecShift, binary},

		TLParen: {PrecCall, func(prec int, p *parser, tok Token, left Expression) Expression {
			call := CallExpr{Func: left}
			for l := p.list(TComma, TRParen); l.next(); {
				call.Args = append(call.Args, p.parseExpression(0))
			}
			return call
		}},
	}
}

var typeParselets map[TokenType]typeParselet

func init() {
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
				p.accept(TIdent)
				ty := p.parseType()
				if ty == nil {
					p.errExpect("type")
				}
				t.Param = append(t.Param, ty)
			}
			t.Ret = p.parseType()
			return t
		},
	}
}

package main

func (p Program) GenProgram(c *Compiler) {
	for _, f := range p {
		f.GenToplevel(c)
	}
}

func (f Function) GenToplevel(c *Compiler) {
	params := make([]IRParam, len(f.Param))
	for i, param := range f.Param {
		params[i].Name = param.Name
		params[i].Ty = param.Ty.Get(c).IRTypeName()
	}

	c.StartFunction(f.Pub, f.Name, params, f.Ret.Get(c).IRTypeName())
	defer c.EndFunction()

	for _, stmt := range f.Code {
		stmt.GenStatement(c)
	}
}

func (d VarDecl) GenStatement(c *Compiler) {
	c.DeclareLocal(d.Name, d.Ty.Get(c))
}
func (d VarDecl) GenToplevel(c *Compiler) {
	c.DeclareGlobal(d.Name, d.Ty.Get(c))
}

func (r ReturnStmt) GenStatement(c *Compiler) {
	v := r.Value.GenExpression(c)
	c.Insn(0, 0, "ret", v)
}

func (e ExprStmt) GenStatement(c *Compiler) {
	e.Expression.GenExpression(c)
}

func (e AssignExpr) GenExpression(c *Compiler) Operand {
	t := e.TypeOf(c).(ConcreteType)
	l := e.L.GenPointer(c)
	r := e.R.GenExpression(c)
	// TODO: make extensible
	c.Insn(0, 0, "store"+t.IRTypeName(), r, l)
	return l
}

func (e CallExpr) GenExpression(c *Compiler) Operand {
	t, ptr := e.typeOf(c)
	var f Operand
	if ptr {
		f = e.Func.GenExpression(c)
	} else {
		f = e.Func.(LValue).GenPointer(c)
	}

	call := CallOperand{f, make([]TypedOperand, len(e.Args))}
	for i, arg := range e.Args {
		// TODO: type-check arguments
		call.Args[i].Ty = arg.TypeOf(c).Concrete().IRTypeName()
		call.Args[i].Op = arg.GenExpression(c)
	}

	if t.Ret == nil {
		c.Insn(0, 0, "call", call)
		return nil
	} else {
		v := c.Temporary()
		c.Insn(v, t.Ret.IRBaseTypeName(), "call", call)
		return v
	}
}

func genLValueExpr(lv LValue, c *Compiler) Operand {
	ty, ok := lv.TypeOf(c).(NumericType)
	if !ok {
		panic("Attempted load of non-numeric type")
	}

	ptr := lv.GenPointer(c)
	op := "load"
	if ty.IRTypeName() != string(ty.IRBaseTypeName()) {
		if ty.(NumericType).Signed() {
			op += "s"
		} else {
			op += "u"
		}
	}
	op += ty.IRTypeName()

	tmp := c.Temporary()
	c.Insn(tmp, ty.IRBaseTypeName(), op, ptr)
	return tmp
}

func (e VarExpr) GenExpression(c *Compiler) Operand {
	return genLValueExpr(e, c)
}
func (e VarExpr) GenPointer(c *Compiler) Operand {
	return c.Variable(string(e)).Loc
}

func (e RefExpr) GenExpression(c *Compiler) Operand {
	return e.V.GenPointer(c)
}

func (e DerefExpr) GenExpression(c *Compiler) Operand {
	return genLValueExpr(e, c)
}
func (e DerefExpr) GenPointer(c *Compiler) Operand {
	return e.V.GenExpression(c)
}

func (e BinaryExpr) GenExpression(c *Compiler) Operand {
	t := e.TypeOf(c).(NumericType)
	l := e.L.GenExpression(c)
	r := e.R.GenExpression(c)
	v := c.Temporary()
	c.Insn(v, t.IRBaseTypeName(), e.Op.Instruction(t), l, r)
	return v
}
func (op BinaryOperator) Instruction(typ NumericType) string {
	switch op {
	case BOpAdd:
		return "add"
	case BOpSub:
		return "sub"
	case BOpMul:
		return "mul"
	case BOpDiv:
		if typ.Signed() {
			return "div"
		} else {
			return "udiv"
		}
	case BOpMod:
		if typ.Signed() {
			return "rem"
		} else {
			return "urem"
		}

	case BOpOr:
		return "or"
	case BOpXor:
		return "xor"
	case BOpAnd:
		return "and"
	case BOpShl:
		return "shl"
	case BOpShr:
		if typ.Signed() {
			return "sar"
		} else {
			return "shr"
		}
	}
	panic("Invalid binary operator")
}

func (e IntegerExpr) GenExpression(c *Compiler) Operand {
	return IRInteger(e)
}
func (e FloatExpr) GenExpression(c *Compiler) Operand {
	panic("TODO")
}
func (e StringExpr) GenExpression(c *Compiler) Operand {
	return c.String(string(e))
}

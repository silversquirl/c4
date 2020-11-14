package main

func (p Program) GenProgram(c *Compiler) {
	for _, tl := range p {
		tl.GenToplevel(c)
	}
}

func (ns NamespaceTL) GenToplevel(c *Compiler) {
	c.StartNamespace(ns.Name)
	for _, tl := range ns.Body {
		tl.GenToplevel(c)
	}
	c.EndNamespace()
}

func (f Function) GenToplevel(c *Compiler) {
	ty := FuncType{}
	ty.Param = make([]ConcreteType, len(f.Param))
	params := make([]IRParam, len(f.Param))
	for i, param := range f.Param {
		params[i].Name = param.Name
		ty.Param[i] = param.Ty.Get(c)
		params[i].Ty = ty.Param[i]
	}

	var ret string
	if f.Ret != nil {
		ty.Ret = f.Ret.Get(c)
		ret = ty.Ret.IRTypeName(c)
	}
	c.StartFunction(f.Pub, f.Name, params, ret)
	c.DeclareGlobal(true, f.Name, ty)

	for _, stmt := range f.Body {
		stmt.GenStatement(c)
	}

	c.EndFunction()
}

func (d VarsDecl) GenStatement(c *Compiler) {
	ty := d.Ty.Get(c)
	for _, name := range d.Names {
		c.DeclareLocal(name, ty)
	}
}
func (d VarsDecl) GenToplevel(c *Compiler) {
	ty := d.Ty.Get(c)
	for _, name := range d.Names {
		c.DeclareGlobal(d.Extern, name, ty)
	}
}

func (t TypeDef) GenToplevel(c *Compiler) {
	c.DefineType(t.Name, t.Ty.Get(c))
}
func (t TypeAlias) GenToplevel(c *Compiler) {
	c.AliasType(t.Name, t.Ty.Get(c))
}

func (i IfStmt) GenStatement(c *Compiler) {
	thenB := c.Block()
	elseB := c.Block()
	endB := c.Block()

	cond := i.Cond.GenExpression(c)
	c.Insn(0, 0, "jnz", cond, thenB, elseB)

	c.StartBlock(thenB)
	for _, stmt := range i.Then {
		stmt.GenStatement(c)
	}
	if !c.ret { // HACK: we shouldn't really access this private field
		c.Insn(0, 0, "jmp", endB)
	}
	c.StartBlock(elseB)
	for _, stmt := range i.Else {
		stmt.GenStatement(c)
	}
	c.StartBlock(endB)
}

func (f ForStmt) GenStatement(c *Compiler) {
	startB := c.Block()
	bodyB := c.Block()
	endB := c.Block()

	c.StartLoop(startB, endB)

	// TODO: scope
	if f.Init != nil {
		f.Init.GenStatement(c)
	}

	c.StartBlock(startB)
	if f.Cond != nil {
		cond := f.Cond.GenExpression(c)
		c.Insn(0, 0, "jnz", cond, bodyB, endB)
	}

	c.StartBlock(bodyB)
	for _, stmt := range f.Body {
		stmt.GenStatement(c)
	}

	if f.Step != nil {
		f.Step.GenExpression(c)
	}
	c.Insn(0, 0, "jmp", startB)
	c.StartBlock(endB)

	c.EndLoop()
}

func (_ BreakStmt) GenStatement(c *Compiler) {
	c.Insn(0, 0, "jmp", c.Loop().End)
	c.StartBlock(c.Block())
}
func (_ ContinueStmt) GenStatement(c *Compiler) {
	c.Insn(0, 0, "jmp", c.Loop().Start)
	c.StartBlock(c.Block())
}

func (r ReturnStmt) GenStatement(c *Compiler) {
	if r.Value != nil {
		v := r.Value.GenExpression(c)
		c.Insn(0, 0, "ret", v)
	} else {
		c.Insn(0, 0, "ret")
	}
}

func (e ExprStmt) GenStatement(c *Compiler) {
	if e.TypeOf(c) != nil {
		panic("Expression returning non-void cannot be used as statement")
	}
	e.Expression.GenExpression(c)
}

func (e AccessExpr) genPointer(c *Compiler) (Operand, Type) {
	lty := e.L.TypeOf(c)
	if ns, ok := lty.(Namespace); ok {
		return Global(ns.Name + e.R), ns.Vars[e.R]
	}

	l := e.L.GenPointer(c)
	for {
		if p, ok := lty.Concrete().(PointerType); ok {
			lty = p.To
			l = genPtrLoad(l, PointerType{}, c)
		} else {
			break
		}
	}

	comp := lty.Concrete().(CompositeType)
	fty := comp.Field(e.R)
	if off := comp.Offset(e.R); off > 0 {
		t := c.Temporary()
		c.Insn(t, 'l', "add", l, IRInt(off))
		return t, fty
	} else {
		return l, fty
	}
}
func (e AccessExpr) GenPointer(c *Compiler) Operand {
	return genLValuePtr(e, c)
}
func (e AccessExpr) GenExpression(c *Compiler) Operand {
	return genLValueExpr(e, c)
}

func (e AssignExpr) GenExpression(c *Compiler) Operand {
	if name, ok := e.L.(VarExpr); ok && name == "_" {
		return e.R.GenExpression(c)
	}

	// TODO: allow storing non-numeric types
	ty := e.typeOf(c).Concrete().(NumericType)
	l := e.L.GenPointer(c)
	r := e.R.GenExpression(c)
	genPtrStore(l, r, ty, c)
	return l
}
func (e MutateExpr) GenExpression(c *Compiler) Operand {
	ty := e.typeOf(c).Concrete().(NumericType)

	l := e.L.GenPointer(c)
	lv := genPtrLoad(l, ty, c)
	r := e.R.GenExpression(c)

	v := e.Op.genExpression(c, lv, r, e.L.TypeOf(c), e.R.TypeOf(c), ty)
	genPtrStore(l, v, ty, c)
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

	call := CallOperand{t.Var, f, make([]TypedOperand, len(e.Args))}
	for i, arg := range e.Args {
		var ty Type
		if i < len(t.Param) {
			ty = t.Param[i]
		} else {
			ty = arg.TypeOf(c)
		}
		call.Args[i].Ty = ty.Concrete().IRTypeName(c)
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

func (e CastExpr) GenExpression(c *Compiler) Operand {
	ty := e.TypeOf(c).Concrete().(NumericType)
	vty := e.V.TypeOf(c).Concrete().(NumericType)

	v := e.V.GenExpression(c)
	// TODO: floating point types
	if vty.Metrics().Size >= ty.Metrics().Size {
		return v
	}

	var insn string
	if vty.Signed() {
		insn = "exts"
	} else {
		insn = "extu"
	}
	insn += vty.IRTypeName(c)

	t := c.Temporary()
	c.Insn(t, ty.IRBaseTypeName(), insn, v)
	return t
}

func genPtrStore(ptr, val Operand, ty NumericType, c *Compiler) {
	// TODO: make extensible
	c.Insn(0, 0, "store"+ty.IRTypeName(c), val, ptr)
}
func genPtrLoad(ptr Operand, ty NumericType, c *Compiler) Operand {
	op := "load"
	if ty.IRTypeName(c) != string(ty.IRBaseTypeName()) {
		if ty.Signed() {
			op += "s"
		} else {
			op += "u"
		}
	}
	op += ty.IRTypeName(c)

	tmp := c.Temporary()
	c.Insn(tmp, ty.IRBaseTypeName(), op, ptr)
	return tmp
}
func genLValueExpr(lv LValue, c *Compiler) Operand {
	ptr, ty := lv.genPointer(c)
	switch ty := ty.Concrete().(type) {
	case NumericType:
		return genPtrLoad(ptr, ty, c)
	default:
		return ptr
	}
}
func genLValuePtr(lv LValue, c *Compiler) Operand {
	v, ty := lv.genPointer(c)
	if _, ok := ty.(ArrayType); ok {
		panic("Cannot reference array field")
	}
	return v
}

func (e VarExpr) genPointer(c *Compiler) (Operand, Type) {
	return c.Variable(string(e)).Loc, c.Variable(string(e)).Ty
}
func (e VarExpr) GenPointer(c *Compiler) Operand {
	return genLValuePtr(e, c)
}
func (e VarExpr) GenExpression(c *Compiler) Operand {
	return genLValueExpr(e, c)
}

func (e RefExpr) GenExpression(c *Compiler) Operand {
	return e.V.GenPointer(c)
}

func (e DerefExpr) genPointer(c *Compiler) (Operand, Type) {
	return e.V.GenExpression(c), e.TypeOf(c)
}
func (e DerefExpr) GenPointer(c *Compiler) Operand {
	return genLValuePtr(e, c)
}
func (e DerefExpr) GenExpression(c *Compiler) Operand {
	return genLValueExpr(e, c)
}

func (e PrefixExpr) GenExpression(c *Compiler) Operand {
	t := e.TypeOf(c).Concrete().(NumericType)
	v := e.V.GenExpression(c)
	tmp := c.Temporary()
	op, arg0 := e.Op.Instruction(c, t)
	if arg0 == nil {
		c.Insn(tmp, t.IRBaseTypeName(), op, v)
	} else {
		c.Insn(tmp, t.IRBaseTypeName(), op, arg0, v)
	}
	return tmp
}

var _ = [1]int{0}[PrefixOperatorMax-5] // Assert correct number of prefix operators
func (op PrefixOperator) Instruction(c *Compiler, ty NumericType) (string, Operand) {
	ity := string(ty.IRBaseTypeName())
	switch op {
	case PrefNot:
		return "ceq" + ity, IRInt(0)
	case PrefInv:
		return "xor", IRInt(-1)
	case PrefNeg:
		return "sub", IRInt(0)
	case PrefPos:
		return "copy", nil
	}
	panic("Invalid prefix operator")
}

func (e BinaryExpr) GenExpression(c *Compiler) Operand {
	ty := e.TypeOf(c).Concrete().(NumericType)
	l := e.L.GenExpression(c)
	r := e.R.GenExpression(c)
	return e.Op.genExpression(c, l, r, e.L.TypeOf(c), e.R.TypeOf(c), ty)
}

func extend(c *Compiler, v Operand, ty NumericType) Operand {
	t := c.Temporary()
	signed := "u"
	if ty.Signed() {
		signed = "s"
	}
	c.Insn(t, 'l', "ext"+signed+ty.IRTypeName(c), v)
	return t
}
func ptrMul(c *Compiler, v Operand, ty PointerType) Operand {
	if ty.To == nil {
		return v
	}
	size := ty.To.Metrics().Size
	if size == 1 {
		return v
	}

	t := c.Temporary()
	c.Insn(t, ty.IRBaseTypeName(), BinMul.Instruction(ty), IRInt(size), v)
	return t
}
func (op BinaryOperator) genExpression(c *Compiler, l, r Operand, lty, rty Type, ty NumericType) Operand {
	lpty, lptr := lty.(PointerType)
	rpty, rptr := rty.(PointerType)

	if lty.IsConcrete() && rty.IsConcrete() {
		lsiz := lty.Concrete().Metrics().Size
		rsiz := rty.Concrete().Metrics().Size
		if lsiz > rsiz {
			r = extend(c, r, rty.Concrete().(NumericType))
		} else if rsiz > lsiz {
			l = extend(c, r, lty.Concrete().(NumericType))
		}
	}

	if !lptr && rptr {
		l = ptrMul(c, l, rpty)
	}
	if lptr && !rptr {
		r = ptrMul(c, r, lpty)
	}

	v := c.Temporary()
	c.Insn(v, ty.IRBaseTypeName(), op.Instruction(ty), l, r)
	return v
}

var _ = [1]int{0}[BinaryOperatorMax-17] // Assert correct number of binary operators
func (op BinaryOperator) Instruction(ty NumericType) string {
	ity := string(ty.IRBaseTypeName())
	switch op {
	case BinAdd:
		return "add"
	case BinSub:
		return "sub"
	case BinMul:
		return "mul"
	case BinDiv:
		if ty.Signed() {
			return "div"
		} else {
			return "udiv"
		}
	case BinMod:
		if ty.Signed() {
			return "rem"
		} else {
			return "urem"
		}

	case BinOr:
		return "or"
	case BinXor:
		return "xor"
	case BinAnd:
		return "and"
	case BinShl:
		return "shl"
	case BinShr:
		if ty.Signed() {
			return "sar"
		} else {
			return "shr"
		}

	case BinCeq:
		return "ceq" + ity
	case BinCne:
		return "cne" + ity
	case BinClt:
		if ty.Signed() {
			return "cslt" + ity
		} else {
			return "cult" + ity
		}
	case BinCgt:
		if ty.Signed() {
			return "csgt" + ity
		} else {
			return "cugt" + ity
		}
	case BinCle:
		if ty.Signed() {
			return "csle" + ity
		} else {
			return "cule" + ity
		}
	case BinCge:
		if ty.Signed() {
			return "csge" + ity
		} else {
			return "cuge" + ity
		}
	}
	panic("Invalid binary operator")
}

func (e BooleanExpr) GenExpression(c *Compiler) Operand {
	t := e.TypeOf(c).Concrete().(NumericType)

	l := e.L.GenExpression(c)
	v := c.Temporary()
	c.Insn(v, t.IRBaseTypeName(), "copy", l)

	longB := c.Block()
	shortB := c.Block()
	e.Op.Emit(c, v, longB, shortB)

	c.StartBlock(longB)
	r := e.R.GenExpression(c)
	c.Insn(v, t.IRBaseTypeName(), "copy", r)

	c.StartBlock(shortB)

	return v
}

var _ = [1]int{0}[BooleanOperatorMax-3] // Assert correct number of binary operators
func (op BooleanOperator) Emit(c *Compiler, v Operand, longB, shortB Block) {
	var a, b Block
	switch op {
	case BoolAnd:
		a, b = longB, shortB
	case BoolOr:
		a, b = shortB, longB
	default:
		panic("Invalid boolean operator")
	}
	c.Insn(0, 0, "jnz", v, a, b)
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
func (e RuneExpr) GenExpression(c *Compiler) Operand {
	return IRInt(int(e))
}

func (p PrimitiveType) GenZero(c *Compiler, loc Operand) {
	c.Insn(0, 0, "store"+p.IRTypeName(c), IRInt(0), loc)
}
func (p PointerType) GenZero(c *Compiler, loc Operand) {
	c.Insn(0, 0, "storel", IRInt(0), loc)
}
func (a ArrayType) GenZero(c *Compiler, loc Operand) {
	off := 0
	m := a.Ty.Metrics()
	for i := 0; i < a.N; i++ {
		op := loc
		if off > 0 {
			t := c.Temporary()
			c.Insn(t, 'l', "add", loc, IRInt(off))
			op = t
		}
		a.Ty.GenZero(c, op)
		off += m.Size
	}
}
func (f FuncType) GenZero(c *Compiler, loc Operand) {
	panic("Attempted to zero a function type")
}

func (ty TypeNameType) GenZero(c *Compiler, loc Operand) {
	ty.get().GenZero(c, loc)
}

func (s StructType) GenZero(c *Compiler, loc Operand) {
	off := 0
	for _, field := range s.compositeType {
		off = -(-off & -field.Ty.Metrics().Align) // Align upwards
		floc := loc
		if off > 0 {
			ftmp := c.Temporary()
			c.Insn(ftmp, 'l', "add", loc, IRInt(off))
			floc = ftmp
		}
		field.Ty.GenZero(c, floc)
		off += field.Ty.Metrics().Size
	}
}

func (u UnionType) GenZero(c *Compiler, loc Operand) {
	var maxTy ConcreteType
	var maxSize int
	for _, field := range u.compositeType {
		size := field.Ty.Metrics().Size
		if size > maxSize {
			maxTy = field.Ty
			maxSize = size
		}
	}
	maxTy.GenZero(c, loc)
}

package main

func (e AssignExpr) TypeOf(c *Compiler) Type {
	ltyp := e.L.TypeOf(c)
	if !ltyp.IsConcrete() {
		panic("Lvalue of non-concrete type")
	}
	rtyp := e.R.TypeOf(c)
	if !Compatible(ltyp, rtyp) {
		panic("Operands of assignment are incompatible")
	}
	return ltyp
}
func (e MutateExpr) TypeOf(c *Compiler) Type {
	return AssignExpr{e.L, BinaryExpr{e.Op, e.L, e.R}}.TypeOf(c)
}

func (e CallExpr) typeOf(c *Compiler) (t FuncType, ptr bool) {
	switch t := e.Func.TypeOf(c).(type) {
	case FuncType:
		return t, false
	case PointerType:
		return t.To.(FuncType), true
	}
	panic("Invalid function type")
}
func (e CallExpr) TypeOf(c *Compiler) Type {
	t, _ := e.typeOf(c)
	return t.Ret
}

func (e VarExpr) TypeOf(c *Compiler) Type {
	return c.Variable(string(e)).Ty
}

func (e RefExpr) TypeOf(c *Compiler) Type {
	return PointerType{e.V.TypeOf(c).(ConcreteType)}
}

func (e DerefExpr) TypeOf(c *Compiler) Type {
	if t, ok := e.V.TypeOf(c).(PointerType); ok {
		return t.To
	} else {
		panic("Dereference of non-pointer type")
	}
}

// FIXME: not and inv require integer types
func (e PrefixExpr) TypeOf(c *Compiler) Type {
	ty := e.V.TypeOf(c)
	if _, ok := ty.Concrete().(NumericType); !ok {
		panic("Operand of prefix expression is of non-numeric type")
	} else {
		return ty
	}
}

// FIXME: all operators other than add, sub, div and mul require integer types
// FIXME: lsh and rsh require their second argument to be an I32 or smaller
func (e BinaryExpr) TypeOf(c *Compiler) Type {
	ltyp := e.L.TypeOf(c)
	rtyp := e.R.TypeOf(c)
	if !Compatible(ltyp, rtyp) {
		panic("Operands of binary expression are incompatible")
	}
	return ltyp
}

func (e BooleanExpr) TypeOf(c *Compiler) Type {
	ltyp := e.L.TypeOf(c)
	rtyp := e.R.TypeOf(c)
	if !Compatible(ltyp, rtyp) {
		panic("Operands of boolean expression are incompatible")
	}
	return ltyp
}

func (_ IntegerExpr) TypeOf(c *Compiler) Type {
	return IntLitType{}
}
func (_ FloatExpr) TypeOf(c *Compiler) Type {
	return FloatLitType{}
}
func (_ StringExpr) TypeOf(c *Compiler) Type {
	// TODO: immutable types
	return PointerType{TypeI8}
}

func (name NamedTypeExpr) Get(c *Compiler) ConcreteType {
	return c.Type(string(name))
}
func (ptr PointerTypeExpr) Get(c *Compiler) ConcreteType {
	return PointerType{ptr.To.Get(c)}
}
func (fun FuncTypeExpr) Get(c *Compiler) ConcreteType {
	params := make([]ConcreteType, len(fun.Param))
	for i, param := range fun.Param {
		params[i] = param.Get(c)
	}
	var ret ConcreteType
	if fun.Ret != nil {
		ret = fun.Ret.Get(c)
	}
	return FuncType{fun.Var, params, ret}
}

func compositeGet(c *Compiler, composite []VarDecl) CompositeType {
	fields := make([]Field, len(composite))
	for i, field := range composite {
		fields[i].Name = field.Name
		fields[i].Ty = field.Ty.Get(c)
	}
	return CompositeType(fields)
}
func (s StructTypeExpr) Get(c *Compiler) ConcreteType {
	return StructType{compositeGet(c, []VarDecl(s))}
}
func (u UnionTypeExpr) Get(c *Compiler) ConcreteType {
	return UnionType{compositeGet(c, []VarDecl(u))}
}

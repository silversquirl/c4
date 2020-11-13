package main

import (
	"fmt"
	"strings"
)

func typeCheck(errCtx string, a, b Type) {
	if !Compatible(a, b) {
		as := a.Format(2)
		bs := b.Format(2)
		var msg string
		if strings.ContainsRune(as, '\n') || strings.ContainsRune(bs, '\n') {
			msg = fmt.Sprintf("Type error in %s:\n\t\t%s\n\tis not\n\t\t%s", errCtx, as, bs)
		} else {
			msg = fmt.Sprintf("Type error in %s: %s is not %s", errCtx, as, bs)
		}
		panic(msg)
	}
}

func (e AccessExpr) TypeOf(c *Compiler) Type {
	lty := e.L.TypeOf(c)
	if ns, ok := lty.(Namespace); ok {
		return ns.Vars[e.R]
	}

	for {
		if p, ok := lty.Concrete().(PointerType); ok {
			lty = p.To
		} else {
			break
		}
	}

	if comp, ok := lty.Concrete().(CompositeType); ok {
		f := comp.Field(e.R)
		if f == nil {
			panic("No such field: " + e.R)
		}
		return f
	}

	panic("Access of non-composite type " + lty.Format(0))
}

func (e AssignExpr) typeOf(c *Compiler) Type {
	if name, ok := e.L.(VarExpr); ok && name == "_" {
		e.R.TypeOf(c)
		return nil
	}

	ltyp := e.L.TypeOf(c)
	if !ltyp.IsConcrete() {
		panic("Lvalue of non-concrete type")
	}
	rtyp := e.R.TypeOf(c)
	typeCheck("assignment", rtyp, ltyp)
	return ltyp
}
func (e MutateExpr) typeOf(c *Compiler) Type {
	return AssignExpr{e.L, BinaryExpr{e.Op, e.L, e.R}}.typeOf(c)
}
func (e AssignExpr) TypeOf(c *Compiler) Type { e.typeOf(c); return nil }
func (e MutateExpr) TypeOf(c *Compiler) Type { e.typeOf(c); return nil }

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
	na, np := len(e.Args), len(t.Param)
	if na < np || (na > np && !t.Var) {
		panic(fmt.Sprintf("Incorrect number of arguments in call to %s: expected %d, got %d", e.Func.Format(0), len(t.Param), len(e.Args)))
	}
	errCtx := "call to " + e.Func.Format(0)
	for i, par := range t.Param {
		typeCheck(errCtx, e.Args[i].TypeOf(c), par)
	}
	return t.Ret
}

func (e VarExpr) TypeOf(c *Compiler) Type {
	return c.Variable(string(e)).Ty
}

func (e RefExpr) TypeOf(c *Compiler) Type {
	return PointerType{e.V.TypeOf(c).Concrete()}
}

func (e DerefExpr) TypeOf(c *Compiler) Type {
	if t, ok := e.V.TypeOf(c).Concrete().(PointerType); ok {
		if t.To == nil {
			panic("Generic pointer may not be dereferenced")
		}
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
	_, lptr := ltyp.Concrete().(PointerType)
	_, rptr := rtyp.Concrete().(PointerType)

	// Ensure ltyp is always the pointer
	if rptr && !lptr {
		lptr, rptr = rptr, lptr
		ltyp, rtyp = rtyp, ltyp
	}

	if lptr && !rptr {
		if _, ok := rtyp.Concrete().(NumericType); !ok {
			typeCheck("binary expression", rtyp, ltyp)
		}
	} else {
		typeCheck("binary expression", rtyp, ltyp)
	}
	return ltyp
}

func (e BooleanExpr) TypeOf(c *Compiler) Type {
	ltyp := e.L.TypeOf(c)
	rtyp := e.R.TypeOf(c)
	typeCheck("boolean expression", rtyp, ltyp)
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
func (_ RuneExpr) TypeOf(c *Compiler) Type {
	return IntLitType{}
}

func (name NamedTypeExpr) Get(c *Compiler) ConcreteType {
	return c.Type(string(name))
}
func (ns NamespaceTypeExpr) Get(c *Compiler) ConcreteType {
	return c.Type(ns...)
}
func (ptr PointerTypeExpr) Get(c *Compiler) ConcreteType {
	if ptr.To == nil {
		return PointerType{}
	}
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

func compositeGet(c *Compiler, composite []VarDecl) compositeType {
	fields := make([]Field, len(composite))
	for i, field := range composite {
		fields[i].Name = field.Name
		fields[i].Ty = field.Ty.Get(c)
	}
	return compositeType(fields)
}
func (s StructTypeExpr) Get(c *Compiler) ConcreteType {
	return StructType{compositeGet(c, []VarDecl(s))}
}
func (u UnionTypeExpr) Get(c *Compiler) ConcreteType {
	return UnionType{compositeGet(c, []VarDecl(u))}
}
